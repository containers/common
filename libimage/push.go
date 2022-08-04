package libimage

import (
	"context"
	"fmt"
	"time"

	dockerArchiveTransport "github.com/containers/image/v5/docker/archive"
	"github.com/containers/image/v5/docker/reference"
	"github.com/containers/image/v5/transports"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/sirupsen/logrus"
)

// PushOptions allows for custommizing image pushes.
type PushOptions struct {
	CopyOptions
	// If true then all images and tags matching a given repository
	// will be pushed. Only supported for the docker transport.
	// Usage of this flag will cause Push() to return a nil []byte.
	AllTags bool
}

// Push pushes the specified source which must refer to an image in the local
// containers storage.  It may or may not have the `containers-storage:`
// prefix.  Use destination to push to a custom destination.  The destination
// can refer to any supported transport.  If not transport is specified, the
// docker transport (i.e., a registry) is implied.  If destination is left
// empty, the docker destination will be extrapolated from the source.
//
// Return storage.ErrImageUnknown if source could not be found in the local
// containers storage.
// Returns the bytes of the copied manifest when pushing a single tag,
// which may be used for digest computation.
// When pushing with AllTags=true then the returned []byte is always nil.
func (r *Runtime) Push(ctx context.Context, source, destination string, options *PushOptions) ([]byte, error) {
	if options == nil {
		options = &PushOptions{}
	}

	// Push the single image
	if !options.AllTags {

		// Look up the local image.  Note that we need to ignore the platform
		// and push what the user specified (containers/podman/issues/10344).
		image, resolvedSource, err := r.LookupImage(source, nil)
		if err != nil {
			return nil, err
		}

		// Make sure we have a proper destination, and parse it into an image
		// reference for copying.
		if destination == "" {
			// Doing an ID check here is tempting but false positives (due
			// to a short partial IDs) are more painful than false
			// negatives.
			destination = resolvedSource
		}

		return pushImage(ctx, image, destination, options, resolvedSource, r)
	}

	// Below handles the AllTags option, for which we have to build a list of
	// all the local images that match the provided repository and then push them.
	//
	// For now, make sure a destination was not specified and get it from the source.
	// This could change in the future, but that gets close to the Copy() functionality.
	if len(destination) != 0 {
		return nil, fmt.Errorf("`destination` should not be specified if using AllTags")
	}

	// Make sure the source repository does not have a tag
	srcNamed, err := reference.ParseNormalizedNamed(source)
	if err != nil {
		return nil, err
	}
	if !reference.IsNameOnly(srcNamed) {
		return nil, fmt.Errorf("can't push with AllTags if source tag is specified")
	}

	logrus.Debugf("Finding all images for source %s", srcNamed.Name())
	listOptions := &ListImagesOptions{}
	srcImages, _ := r.ListImages(ctx, []string{srcNamed.Name()}, listOptions)

	// Push each tag for every image in the list
	for _, img := range srcImages {
		namedTagged, err := img.NamedTaggedRepoTags()
		if err != nil {
			return nil, err
		}
		for _, n := range namedTagged {
			// Filter on repo name again to avoid pushing an image that matches
			// the source image ID but has a different repository than the source
			currentNamed, err := reference.ParseNormalizedNamed(n.Name())
			if err != nil {
				return nil, err
			}
			if reference.Path(currentNamed) == reference.Path(srcNamed) {
				// Have to use Sprintf because pushImage expects a string
				destWithTag := fmt.Sprintf("%s:%s", source, n.Tag())
				_, err := pushImage(ctx, img, destWithTag, options, "", r)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	return nil, nil
}

// pushImage sends a single image to be copied to the destination
func pushImage(ctx context.Context, image *Image, destination string, options *PushOptions, resolvedSource string, r *Runtime) ([]byte, error) {
	srcRef, err := image.StorageReference()
	if err != nil {
		return nil, err
	}

	logrus.Debugf("Pushing image %s to %s", transports.ImageName(srcRef), destination)

	destRef, err := alltransports.ParseImageName(destination)
	if err != nil {
		// If the input does not include a transport assume it refers
		// to a registry.
		dockerRef, dockerErr := alltransports.ParseImageName("docker://" + destination)
		if dockerErr != nil {
			return nil, err
		}
		destRef = dockerRef
	}

	if r.eventChannel != nil {
		defer r.writeEvent(&Event{ID: image.ID(), Name: destination, Time: time.Now(), Type: EventTypeImagePush})
	}

	// Buildah compat: Make sure to tag the destination image if it's a
	// Docker archive. This way, we preserve the image name.
	if destRef.Transport().Name() == dockerArchiveTransport.Transport.Name() {
		if named, err := reference.ParseNamed(resolvedSource); err == nil {
			tagged, isTagged := named.(reference.NamedTagged)
			if isTagged {
				options.dockerArchiveAdditionalTags = []reference.NamedTagged{tagged}
			}
		}
	}

	c, err := r.newCopier(&options.CopyOptions)
	if err != nil {
		return nil, err
	}

	defer c.close()

	return c.copy(ctx, srcRef, destRef)
}
