package libimage

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFilterNames(t *testing.T) {

	regs := []string{
		"localhost",
		"docker.io",
		"foo.bar",
		"registry.access.redhat.com",
	}
	registries := make([]interface{}, len(regs))
	for i, v := range regs {
		registries[i] = v
	}

	names := []string{
		"docker.io/library/alpine:latest",
		"docker.io/library/busybox:latest",
		"docker.io/library/golang:1.13",
		"docker.io/library/golang:1.16",
		"docker.io/library/hello-world:latest",
		"docker.io/library/nginx:latest",
		"docker.io/library/traefik:2.2",
		"docker.io/syncthing/syncthing:1.18.4",
		"localhost/dan:latest",
		"localhost/darkside-image:latest",
		"localhost/myimage:latest",
		"localhost/mysystemd:latest",
		"localhost/restapi_app:latest",
		"localhost/restapi_backend:latest",
		"<none>:<none>",
		"quay.io/libpod/testimage:20210610",
		"quay.io/rhatdan/myimage:latest",
		"registry.access.redhat.com/ubi8-init:latest",
		"registry.access.redhat.com/ubi8:latest",
		"registry.access.redhat.com/ubi8-micro:latest",
		"registry.fedoraproject.org/fedora:latest",
	}

	for _, test := range []struct {
		filter  string
		matches int
	}{
		{"alpine", 1},
		{"alpine*", 1},
		{"localhost/restapi_app:latest", 1},
		{"golang", 2},
		{"golang:1.13", 1},
		{"ubi8", 1},
		{"ubi8*", 3},
		{"myimage*", 1},
		{"foo*", 0},
		{"*box", 1},
		{"my*", 2},
		{"*", len(names)},
		{"*:latest", 15},
		{"localhost/mysystemd", 1},
	} {
		filter := setupFilter(test.filter)
		var imgs []string
		for _, name := range names {
			if ok, _ := filterNames(filter, []string{name}, registries); ok {
				imgs = append(imgs, name)
			}
		}

		fmt.Println(test.filter)
		require.Len(t, imgs, test.matches)
	}
}
