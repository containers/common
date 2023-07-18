% containers-connections.conf(5)

## NAME
containers-connections.conf - configuration file for remote connections to the API service

## DESCRIPTION
The connections.conf file configure the service API destinations. By default, root user reads from **/etc/containers/connections.conf**, while rootless user read from **$HOME/.config/containers/connections.conf**.

## FORMAT
**[{service_name}]**
URI to access the API service
**uri="ssh://user@production.example.com/run/user/1001/podman/podman.sock"**

  Example URIs:

- **rootless local**  - unix://run/user/1000/podman/podman.sock
- **rootless remote** - ssh://user@engineering.lab.company.com/run/user/1000/podman/podman.sock
- **rootful local**  - unix://run/podman/podman.sock
- **rootful remote** - ssh://root@10.10.1.136:22/run/podman/podman.sock

**identity="~/.ssh/id_rsa**

Path to file containing ssh identity key

## EXAMPLE
[hoge]
uri=unix://run/user/1000/podman/podman.sock

[fuga]
uri=ssh://user@engineering.lab.company.com/run/user/1000/podman/podman.sock
identity="~/.ssh/id_rsa"

## HISTORY
June 2023, Originally compiled by Toshiki Sonoda <sonoda.toshiki@fujitsu.com>
