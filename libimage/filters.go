//go:build !remote

package libimage

import (
	"context"
	"errors"
	"fmt"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"

	filtersPkg "github.com/containers/common/pkg/filters"
	"github.com/containers/common/pkg/timetype"
	"github.com/containers/image/v5/docker/reference"
	"github.com/sirupsen/logrus"
)

// filterFunc is a prototype for a positive image filter.  Returning `true`
// indicates that the image matches the criteria.
type filterFunc func(*Image, *layerTree) (bool, error)

// referenceFilterFunc is a prototype for a filter that returns a list of
// references. The first return value indicates whether the image matches the
// criteria. The second return value is a list of names that match the
// criteria. The third return value is an error.
type referenceFilterFunc func(*Image) (bool, []string, error)

type compiledFilters struct {
	filters         map[string][]filterFunc
	referenceFilter referenceFilterFunc
	needsLayerTree  bool
}

// Apply the specified filters.  All filters of each key must apply.
// tree must be provided if compileImageFilters indicated it is necessary.
// WARNING: Application of referenceFilter sets the image names to matched names, but this only affects the values in memory, they are not written to storage.
func (i *Image) applyFilters(ctx context.Context, f *compiledFilters, tree *layerTree) (bool, error) {
	for key := range f.filters {
		for _, filter := range f.filters[key] {
			matches, err := filter(i, tree)
			if err != nil {
				// Some images may have been corrupted in the
				// meantime, so do an extra check and make the
				// error non-fatal (see containers/podman/issues/12582).
				if errCorrupted := i.isCorrupted(ctx, ""); errCorrupted != nil {
					logrus.Error(errCorrupted.Error())
					return false, nil
				}
				return false, err
			}
			// If any filter within a group doesn't match, return false
			if !matches {
				return false, nil
			}
		}
	}
	if f.referenceFilter != nil {
		referenceMatch, names, err := f.referenceFilter(i)
		if err != nil {
			// Some images may have been corrupted in the
			// meantime, so do an extra check and make the
			// error non-fatal (see containers/podman/issues/12582).
			if errCorrupted := i.isCorrupted(ctx, ""); errCorrupted != nil {
				logrus.Error(errCorrupted.Error())
				return false, nil
			}
			return false, err
		}
		if !referenceMatch {
			return false, nil
		}
		if len(names) > 0 {
			i.setEphemeralNames(names)
		}
	}
	return true, nil
}

// filterImages returns a slice of images which are passing all specified
// filters.
// tree must be provided if compileImageFilters indicated it is necessary.
// WARNING: Application of referenceFilter sets the image names to matched names, but this only affects the values in memory, they are not written to storage.
func (r *Runtime) filterImages(ctx context.Context, images []*Image, filters *compiledFilters, tree *layerTree) ([]*Image, error) {
	result := []*Image{}
	for i := range images {
		match, err := images[i].applyFilters(ctx, filters, tree)
		if err != nil {
			return nil, err
		}
		if match {
			result = append(result, images[i])
		}
	}
	return result, nil
}

// compileImageFilters creates `filterFunc`s for the specified filters.  The
// required format is `key=value` with the following supported keys:
//
//	after, since, before, containers, dangling, id, label, readonly, reference, intermediate
//
// compileImageFilters returns: compiled filters, if LayerTree is needed, error
func (r *Runtime) compileImageFilters(ctx context.Context, options *ListImagesOptions) (*compiledFilters, error) {
	logrus.Tracef("Parsing image filters %s", options.Filters)
	if len(options.Filters) == 0 {
		return &compiledFilters{}, nil
	}

	filterInvalidValue := `invalid image filter %q: must be in the format "filter=value or filter!=value"`

	var wantedReferenceMatches, unwantedReferenceMatches []string
	cf := compiledFilters{
		filters:        map[string][]filterFunc{},
		needsLayerTree: false,
	}
	duplicate := map[string]string{}
	for _, f := range options.Filters {
		var key, value string
		var filter filterFunc
		negate := false
		split := strings.SplitN(f, "!=", 2)
		if len(split) == 2 {
			negate = true
		} else {
			split = strings.SplitN(f, "=", 2)
			if len(split) != 2 {
				return nil, fmt.Errorf(filterInvalidValue, f)
			}
		}

		key = split[0]
		value = split[1]
		switch key {
		case "after", "since":
			img, err := r.time(key, value)
			if err != nil {
				return nil, err
			}
			key = "since"
			filter = filterAfter(img.Created())

		case "before":
			img, err := r.time(key, value)
			if err != nil {
				return nil, err
			}
			filter = filterBefore(img.Created())

		case "containers":
			if err := r.containers(duplicate, key, value, options.IsExternalContainerFunc); err != nil {
				return nil, err
			}
			filter = filterContainers(value, options.IsExternalContainerFunc)

		case "dangling":
			dangling, err := r.bool(duplicate, key, value)
			if err != nil {
				return nil, err
			}
			cf.needsLayerTree = true
			filter = filterDangling(ctx, dangling)

		case "id":
			filter = filterID(value)

		case "digest":
			f, err := filterDigest(value)
			if err != nil {
				return nil, err
			}
			filter = f

		case "intermediate":
			intermediate, err := r.bool(duplicate, key, value)
			if err != nil {
				return nil, err
			}
			cf.needsLayerTree = true
			filter = filterIntermediate(ctx, intermediate)

		case "label":
			filter = filterLabel(ctx, value)
		case "readonly":
			readOnly, err := r.bool(duplicate, key, value)
			if err != nil {
				return nil, err
			}
			filter = filterReadOnly(readOnly)

		case "manifest":
			manifest, err := r.bool(duplicate, key, value)
			if err != nil {
				return nil, err
			}
			filter = filterManifest(ctx, manifest)

		case "reference":
			if negate {
				unwantedReferenceMatches = append(unwantedReferenceMatches, value)
			} else {
				wantedReferenceMatches = append(wantedReferenceMatches, value)
			}
			continue

		case "until":
			until, err := r.until(value)
			if err != nil {
				return nil, err
			}
			filter = filterBefore(until)

		default:
			return nil, fmt.Errorf(filterInvalidValue, key)
		}
		if negate {
			filter = negateFilter(filter)
		}
		cf.filters[key] = append(cf.filters[key], filter)
	}

	// reference filters is a special case as it does an OR for positive matches
	// and an AND logic for negative matches and the filter function type is different.
	cf.referenceFilter = filterReferences(r, wantedReferenceMatches, unwantedReferenceMatches)
	return &cf, nil
}

func negateFilter(f filterFunc) filterFunc {
	return func(img *Image, tree *layerTree) (bool, error) {
		b, err := f(img, tree)
		return !b, err
	}
}

func (r *Runtime) containers(duplicate map[string]string, key, value string, externalFunc IsExternalContainerFunc) error {
	if exists, ok := duplicate[key]; ok && exists != value {
		return fmt.Errorf("specifying %q filter more than once with different values is not supported", key)
	}
	duplicate[key] = value
	switch value {
	case "false", "true":
	case "external":
		if externalFunc == nil {
			return errors.New("libimage error: external containers filter without callback")
		}
	default:
		return fmt.Errorf("unsupported value %q for containers filter", value)
	}
	return nil
}

func (r *Runtime) until(value string) (time.Time, error) {
	var until time.Time
	ts, err := timetype.GetTimestamp(value, time.Now())
	if err != nil {
		return until, err
	}
	seconds, nanoseconds, err := timetype.ParseTimestamps(ts, 0)
	if err != nil {
		return until, err
	}
	return time.Unix(seconds, nanoseconds), nil
}

func (r *Runtime) time(key, value string) (*Image, error) {
	img, _, err := r.LookupImage(value, nil)
	if err != nil {
		return nil, fmt.Errorf("could not find local image for filter %q=%q: %w", key, value, err)
	}
	return img, nil
}

func (r *Runtime) bool(duplicate map[string]string, key, value string) (bool, error) {
	if exists, ok := duplicate[key]; ok && exists != value {
		return false, fmt.Errorf("specifying %q filter more than once with different values is not supported", key)
	}
	duplicate[key] = value
	set, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("non-boolean value %q for %s filter: %w", key, value, err)
	}
	return set, nil
}

// filterManifest filters whether or not the image is a manifest list
func filterManifest(ctx context.Context, value bool) filterFunc {
	return func(img *Image, _ *layerTree) (bool, error) {
		isManifestList, err := img.IsManifestList(ctx)
		if err != nil {
			return false, err
		}
		return isManifestList == value, nil
	}
}

// filterReferences creates a reference filter for matching the specified wantedReferenceMatches value (OR logic)
// and for matching the unwantedReferenceMatches values (AND logic)
func filterReferences(r *Runtime, wantedReferenceMatches, unwantedReferenceMatches []string) referenceFilterFunc {
	return func(img *Image) (bool, []string, error) {
		// Empty reference filters, return true
		if len(wantedReferenceMatches) == 0 && len(unwantedReferenceMatches) == 0 {
			return true, nil, nil
		}

		// Go through the unwanted matches first
		// TODO 6.0 podman: remove unwanted matches from the output names. https://github.com/containers/common/pull/2413#discussion_r2031749013
		for _, value := range unwantedReferenceMatches {
			names, err := getMatchedImageNames(r, img, value)
			if err != nil {
				return false, nil, err
			}
			if len(names) > 0 {
				return false, nil, nil
			}
		}

		namesThatMatch := slices.Clone(img.Names())
		// If there are no wanted match filters, then return true for the image
		// that don't march matched the unwanted filters.
		if len(wantedReferenceMatches) == 0 {
			return true, namesThatMatch, nil
		}

		matchedNames := map[string]struct{}{}

		// If the wanted reference is RepoDigest and image match. All names of image are returned.
		isRepoDigest := false

		for _, value := range wantedReferenceMatches {
			names, err := getMatchedImageNames(r, img, value)
			if err != nil {
				return false, nil, err
			}

			for name := range names {
				repoDigests, err := img.RepoDigests()
				if err != nil {
					return false, nil, err
				}
				for _, repoDigest := range repoDigests {
					if name == repoDigest {
						isRepoDigest = true
						break
					}
				}

				if isRepoDigest {
					break
				}
				matchedNames[name] = struct{}{}
			}

			if isRepoDigest {
				break
			}
		}

		if isRepoDigest {
			return true, namesThatMatch, nil
		}

		if len(matchedNames) > 0 {
			// Removes non-compliant names from image names
			namesThatMatch = slices.DeleteFunc(namesThatMatch, func(name string) bool {
				_, ok := matchedNames[name]
				return !ok
			})
			return true, namesThatMatch, nil
		}

		return false, nil, nil
	}
}

// getMatchedImageNames returns a set of matching image names that match the specified filter value, or an empty list if the image does not match the filter.
func getMatchedImageNames(r *Runtime, img *Image, value string) (map[string]struct{}, error) {
	lookedUp, resolvedName, _ := r.LookupImage(value, nil)
	if lookedUp != nil {
		if lookedUp.ID() == img.ID() {
			return map[string]struct{}{resolvedName: {}}, nil
		}
	}

	refs, err := img.NamesReferences()
	if err != nil {
		return nil, err
	}

	resolvedNames := map[string]struct{}{}
	for _, ref := range refs {
		refString := ref.String() // FQN with tag/digest
		candidates := []string{refString}

		// Split the reference into 3 components (twice if digested/tagged):
		// 1) Fully-qualified reference
		// 2) Without domain
		// 3) Without domain and path
		if named, isNamed := ref.(reference.Named); isNamed {
			candidates = append(candidates,
				reference.Path(named),                           // path/name without tag/digest (Path() removes it)
				refString[strings.LastIndex(refString, "/")+1:]) // name with tag/digest

			trimmedString := reference.TrimNamed(named).String()
			if refString != trimmedString {
				tagOrDigest := refString[len(trimmedString):]
				candidates = append(candidates,
					trimmedString,                     // FQN without tag/digest
					reference.Path(named)+tagOrDigest, // path/name with tag/digest
					trimmedString[strings.LastIndex(trimmedString, "/")+1:]) // name without tag/digest
			}
		}

		for _, candidate := range candidates {
			// path.Match() is also used by Docker's reference.FamiliarMatch().
			matched, _ := path.Match(value, candidate)
			if matched {
				resolvedNames[refString] = struct{}{}
				break
			}
		}
	}
	return resolvedNames, nil
}

// filterLabel creates a label for matching the specified value.
func filterLabel(ctx context.Context, value string) filterFunc {
	return func(img *Image, _ *layerTree) (bool, error) {
		labels, err := img.Labels(ctx)
		if err != nil {
			return false, err
		}
		return filtersPkg.MatchLabelFilters([]string{value}, labels), nil
	}
}

// filterAfter creates an after filter for matching the specified value.
func filterAfter(value time.Time) filterFunc {
	return func(img *Image, _ *layerTree) (bool, error) {
		return img.Created().After(value), nil
	}
}

// filterBefore creates a before filter for matching the specified value.
func filterBefore(value time.Time) filterFunc {
	return func(img *Image, _ *layerTree) (bool, error) {
		return img.Created().Before(value), nil
	}
}

// filterReadOnly creates a readonly filter for matching the specified value.
func filterReadOnly(value bool) filterFunc {
	return func(img *Image, _ *layerTree) (bool, error) {
		return img.IsReadOnly() == value, nil
	}
}

// filterContainers creates a container filter for matching the specified value.
func filterContainers(value string, fn IsExternalContainerFunc) filterFunc {
	return func(img *Image, _ *layerTree) (bool, error) {
		ctrs, err := img.Containers()
		if err != nil {
			return false, err
		}
		if value != "external" {
			boolValue := value == "true"
			return (len(ctrs) > 0) == boolValue, nil
		}

		// Check whether all associated containers are external ones.
		for _, c := range ctrs {
			isExternal, err := fn(c)
			if err != nil {
				return false, fmt.Errorf("checking if %s is an external container in filter: %w", c, err)
			}
			if !isExternal {
				return isExternal, nil
			}
		}
		return true, nil
	}
}

// filterDangling creates a dangling filter for matching the specified value.
func filterDangling(ctx context.Context, value bool) filterFunc {
	return func(img *Image, tree *layerTree) (bool, error) {
		isDangling, err := img.isDangling(ctx, tree)
		if err != nil {
			return false, err
		}
		return isDangling == value, nil
	}
}

// filterID creates an image-ID filter for matching the specified value.
func filterID(value string) filterFunc {
	return func(img *Image, _ *layerTree) (bool, error) {
		return strings.HasPrefix(img.ID(), value), nil
	}
}

// filterDigest creates a digest filter for matching the specified value.
func filterDigest(value string) (filterFunc, error) {
	if !strings.HasPrefix(value, "sha256:") {
		return nil, fmt.Errorf("invalid value %q for digest filter", value)
	}
	return func(img *Image, _ *layerTree) (bool, error) {
		return img.containsDigestPrefix(value), nil
	}, nil
}

// filterIntermediate creates an intermediate filter for images.  An image is
// considered to be an intermediate image if it is dangling (i.e., no tags) and
// has no children (i.e., no other image depends on it).
func filterIntermediate(ctx context.Context, value bool) filterFunc {
	return func(img *Image, tree *layerTree) (bool, error) {
		isIntermediate, err := img.isIntermediate(ctx, tree)
		if err != nil {
			return false, err
		}
		return isIntermediate == value, nil
	}
}
