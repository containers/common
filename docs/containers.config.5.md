% containers.conf(5) Container runtime tools configuration file

# NAME
containers.conf - The containers configuration file specifies all of the available configuration options and command-line flags for container runtime tools like Podman & Buildah, but in a TOML format that can be easily modified and versioned.

# DESCRIPTION
The containers configuration file specifies all of the available configuration options and command-line flags for container runtime tools like Podman & Buildah, but in a TOML format that can be easily modified and versioned.

# FORMAT
The [TOML format][toml] is used as the encoding of the configuration file. Every option is nested under its table. No bare options are used. The format of TOML can be simplified to:

    [table1]
    option = value

    [table2]
    option = value

    [table3]
    option = value

    [table3.subtable1]
    option = value

## CONTAINERS TABLE
The containers table contains settings pertaining to the OCI runtime that can configure and manage the OCI runtime.

**additional_devices**=[]
  List of additional devices. Specified as "<device-on-host>:<device-on-container>:<permissions>", for example: "--additional-devices=/dev/sdc:/dev/xvdc:rwm".

**apparmor_profile**=""
  Used to change the name of the default AppArmor profile of container engines. The default profile name is "container-default".

**cgroup_manager**="systemd"
 The cgroup management implementation used for the runtime. Supports cgroupfs and systemd.

**default_capabilities**=[]
  List of default capabilities for containers.

  The default list is:
```
  default_capabilities = [
        "AUDIT_WRITE",
	      "CHOWN",
	      "DAC_OVERRIDE",
	      "FOWNER",
	      "FSETID",
	      "KILL",
	      "MKNOD",
	      "NET_BIND_SERVICE",
	      "NET_RAW",
	      "SETGID",
	      "SETPCAP",
	      "SETUID",
	      "SYS_CHROOT",
  ]
```

**default_sysctls**=[]
  List of default sysctls.

**default_ulimits**=[]
  A list of ulimits to be set in containers by default, specified as "<ulimit name>=<soft limit>:<hard limit>", for example:"nofile=1024:2048".

**env**=["PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"]
  Environment variable list for the container process, used for passing environment variables to the container.

**hooks_dir**=["/etc/containers/oci/hooks.d", ...]
  Path to the OCI hooks directories for automatically executed hooks.

**http_proxy**=[]"`
  The proxy environment variable list to apply to container process

**init**=false
  Run an init inside the container that forwards signals and reaps processes.

**label**="true|false"
  Indicates whether the containers should use label separation.

**log_size_max**=-1
  Maximum size allowed for the container's log file. Negative numbers indicate that no size limit is imposed. If it is positive, it must be >= 8192 to match/exceed conmon's read buffer. The file is truncated and re-opened so the limit is never exceeded.

**pids_limit**=1024
  Maximum number of processes allowed in a container. 0 indicates that no limit is imposed.

**seccomp_profile**="/usr/share/containers/seccomp.json"
  Path to the seccomp.json profile which is used as the default seccomp profile for the runtime.

**shm_size**=64m
  Size of `/dev/shm`. The format is `<number><unit>`. `number` must be greater than `0`. Unit is optional and can be `b` (bytes), `k` (kilobytes), `m`(megabytes), or `g` (gigabytes). If you omit the unit, the system uses bytes. If you omit the size entirely, the system uses `64m`.

**signature_policy_path**=""
  The path to a signature policy to use for validating images. If left empty, the containers/image default signature policy will be used.

## NETWORK TABLE
The `network` table containers settings pertaining to the management of CNI plugins.

**cni_plugin_dirs**=["/opt/cni/bin/",]
  List of paths to directories where CNI plugin binaries are located.

**default_network**="podman"
  The network name of the default CNI network to attach pods to.

**network_config_dir**="/etc/cni/net.d/"
  Path to the directory where CNI configuration files are located.

## LIBPOD TABLE
The`libpod` table contains configuration options used to set up a libpod runtime

**conmon_env_vars**=""
  Environment variables to pass into Conmon

**conmon_path**=""
  Paths to search for the conmon container manager binary. If the paths are empty or no valid path was found, then the `$PATH` environment variable will be used as the fallback.

  The default list is
```
conmon_path=[
		"/usr/libexec/podman/conmon",
		"/usr/local/libexec/podman/conmon",
		"/usr/local/lib/podman/conmon",
		"/usr/bin/conmon",
		"/usr/sbin/conmon",
		"/usr/local/bin/conmon",
		"/usr/local/sbin/conmon",
		"/run/current-system/sw/bin/conmon",
]
```

**detach_keys**=""
  Keys sequence used for detaching a container

**enable_port_reservation**=true
  Determines whether libpod will reserve ports on the host when they are forwarded to containers. When enabled, when ports are forwarded to containers, they are held open by conmon as long as the container is running, ensuring that they cannot be reused by other
  programs on the host. However, this can cause significant memory usage if
  a container has many ports forwarded to it. Disabling this can save
  memory.

**events_logfile_path**=""
  Path to the file where the events log is stored. The default is

**events_logger**=""
  Default method to use when logging events. Valid values are "file", "journald", and "none".

**image_default_transport**=""
  Default transport method for pulling and pushing images

**infra_command**=""
  Command to run the infra container

**infra_image** = ""
  Infra (pause) container image name for pod infra containers.  When running a pod, we
  start a `pause` process in a container to hold open the namespaces associated with the
  pod.  This container and process, basically sleep/pause for the lifetime of the pod.

**init_path**=""
  Path to the container-init binary, which forwards signals and reaps processes within containers.  Note that the container-init binary will only be used when the `--init` for podman-create and podman-run is set.

**lock_type**=""
  Specify the locking mechanism to use; valid values are "shm" and "file".  Change the default only if you are sure of what you are doing, in general "file" is useful only on platforms where cgo is not available for using the faster "shm" lock type.  You may need to run "podman system renumber" after you change the lock type.

**namespace**=""
  Default libpod namespace. If libpod is joined to a namespace, it will see only containers and pods
  that were created in the same namespace, and will create new containers and pods in that namespace.
  The default namespace is "", which corresponds to no namespace. When no namespace is set, all
  containers and pods are visible.

**network_cmd_path**=""
  Path to the command binary to use for setting up a network.  It is currently only used for setting up
  a slirp4netns network.  If "" is used then the binary is looked up using the $PATH environment variable.

**no_pivot_root**=""
  Whether to use chroot instead of pivot_root in the runtime

**num_locks**=""
  Number of locks available for containers and pods. Each created container or pod consumes one lock.
  The default number available is 2048.
  If this is changed, a lock renumbering must be performed, using the `podman system renumber` command.

**runtime**=""
  Default OCI runtime to use if nothing is specified in **runtimes**

**runtimes**
  For each OCI runtime, specify a list of paths to look for.  The first one found is used. If the paths are empty or no valid path was found, then the `$PATH` environment variable will be used as the fallback.

**runtime_path**=[]
  RuntimePath is the path to OCI runtime binary for launching containers. The first path pointing to a valid file will be used This is used only when there are no OCIRuntime/OCIRuntimes defined.  It is used only to be backward compatible with older versions of Podman.

**runtime_supports_json**=[]"`
  The list of the OCI runtimes that support `--format=json`.

**runtime_supports_nocgroups**=[]"
  Theist of OCI runtimes that support running containers without CGroups.

**static_dir**=""
  Directory for persistent libpod files (database, etc)
  By default this will be configured relative to where containers/storage
  stores containers

**tmp_dir**=""
  The path to a temporary directory to store per-boot container
  Must be a tmpfs (wiped after reboot)

**volume_path**=""
  Directory where named volumes will be created in using the default volume driver.
  By default this will be configured relative to where containers/storage stores containers.
  This convention is followed by the default volume driver, but may not be by other drivers.

# SEE ALSO
containers-storage.conf(5), containers-policy.json(5), containers-registries.conf(5)

[toml]: https://github.com/toml-lang/toml