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

## CONTAINERS TABLE
The containers table contains settings pertaining to the OCI runtime that can configure and manage the OCI runtime.

**default_ulimits**=[]
  A list of ulimits to be set in containers by default, specified as "<ulimit name>=<soft limit>:<hard limit>", for example:"nofile=1024:2048".

**env**=["PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"]
  Environment variable list for the container process, used for passing environment variables to the container.

**selinux**=true
  If false, SELinux will not be used for pod separation on the host.

**seccomp_profile**="/usr/share/containers/seccomp.json"
  Path to the seccomp.json profile which is used as the default seccomp profile for the runtime.

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

**additional_devices**=[]
  List of additional devices. Specified as "<device-on-host>:<device-on-container>:<permissions>", for example: "--additional-devices=/dev/sdc:/dev/xvdc:rwm".

**hooks_dir**=["/etc/containers/oci/hooks.d", ...]
  Path to the OCI hooks directories for automatically executed hooks.

**pids_limit**=1024
  Maximum number of processes allowed in a container. 0 indicates that no limit is imposed..

**log_size_max**=-1
  Maximum size allowed for the container's log file. Negative numbers indicate that no size limit is imposed. If it is positive, it must be >= 8192 to match/exceed conmon's read buffer. The file is truncated and re-opened so the limit is never exceeded.

**shm_size**=64m
  Size of `/dev/shm`. The format is `<number><unit>`. `number` must be greater than `0`. Unit is optional and can be `b` (bytes), `k` (kilobytes), `m`(megabytes), or `g` (gigabytes). If you omit the unit, the system uses bytes. If you omit the size entirely, the system uses `64m`.

**init**=false
  Run an init inside the container that forwards signals and reaps processes.

## NETWORK TABLE
The `network` table containers settings pertaining to the management of CNI plugins.

**network_dir**="/etc/cni/net.d/"
  Path to the directory where CNI configuration files are located.

**plugin_dirs**=["/opt/cni/bin/",]
  List of paths to directories where CNI plugin binaries are located.

# SEE ALSO
containers-storage.conf(5), containers-policy.json(5), containers-registries.conf(5)

[toml]: https://github.com/toml-lang/toml