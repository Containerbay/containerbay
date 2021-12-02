<h1 align="center">
  <img src=https://user-images.githubusercontent.com/2420543/144125454-c07ebb53-50af-4495-9214-47bb1b0c415b.png width=72> 
  <br>
  Containerbay
<br>

</h1>

<h3 align="center">Web gateway for OCI artifacts</h3>
<p align="center">
  <a href="https://opensource.org/licenses/">
    <img src="https://img.shields.io/badge/licence-GPL3-brightgreen"
         alt="license">
  </a>
  <a href="https://github.com/mudler/containerbay/issues"><img src="https://img.shields.io/github/issues/mudler/containerbay"></a>
  <img src="https://img.shields.io/badge/made%20with-Go-blue">
  <img src="https://goreportcard.com/badge/github.com/mudler/containerbay" alt="go report card" />
</p>

<p align="center">
	 <br>
    Container images gateway browser and indexer<br>
    <b>Website static server</b> -  <b>Reverse Container image browser</b>
</p>

Containerbay allows to serve OCI container artifacts as static websites and browse them from curl or your browser. Works also with MagicDNS(tm).

# :notebook: Example

```bash
curl https://containerbay.io/quay.io/mudler/containerbay:website/
```

Some notable examples that you can just browse right away:

- [Our example page here](https://containerbay.io/ghcr.io/containerbay/containerbay.io:latest/)  hosted on [ghcr.io](https://ghcr.io/containerbay/containerbay.io)!
- [openSUSE](https://containerbay.io/docker.io/opensuse/leap/)
- [Alpine](https://containerbay.io/docker.io/library/alpine/)
- [Alpine (mirror)](https://containerbay.io/mirror.gcr.io/library/alpine/etc/)

If there is no index page, it will fallback to list all the present files, so it can be used to browse also already existing container images content

# :computer: Usage

Containerbay can be used to explore container images with curl, but can also be used to bind static domain, or run standalone servers that bind to a single image reference

## API

Point your browser or either `curl` to access the container's content in this form `host/registry/org/image_name:tag/`, the host should be pointed to a containerbay instance ( like `containerbay.io` ):

```bash
curl https://containerbay.io/docker.io/opensuse/leap/etc/os-release
curl https://containerbay.io/docker.io/opensuse/leap@sha256:b603e69d71c9d9b3ec1fcd89d2db2f3c82d757e8a724a8602d6514dc4c77b1cb/
```

## MagicDNS(tm)

When Containerbay is running, it accepts container images from subdomains in the following format `registry.org.image_name.tag.magicdns.io`

For example, by using `containerbay.io`
```bash
curl http://docker.io.library.alpine.latest.containerbay.io/etc/os-release
```

will return `/etc/os-release` from `alpine:latest`.

## Bind to a custom domain

Containerbay can associate a custom domain to a container image. In this way you can have images containing static HTML files and use it to serve a subdomain or a top level dns. See as an [example repository](https://github.com/containerbay/containerbay.io).

Point your DNS to the containerbay instance via `A` or `CNAME` and add a corresponding `TXT` record.

In the `TXT` record, write the image you want to serve:
```
containerbay=library/alpine:latest
```

## Serve a static website

Containerbay can be used to deploy static website.

There is available a `pack` command which is a helper in order to create tar archives from files that can be loaded from docker in order to be pushed over a registry, for instance, if you have a folder with a index.html:

```bash
# Create output.tar, and include api/public/index.html. The container image will be named quay.io/mudler/containerbay:website
containerbay pack --destination output.tar quay.io/mudler/containerbay:website api/public/index.html
# Load the image to the docker daemon
docker load -i output.tar
# Push the container image
docker push quay.io/mudler/containerbay:website
```

`containerbay pack` accepts a destination output with the `--destination` flag, first argument is the container image

Alternatively you can also just use `docker`, or `podman` or your favorite container builder, check the `website` folder for an example.

# :running: Deploy

Containerbay is currently a service deployed at [containerbay.io](https://containerbay.io). 

You can although choose to deploy containerbay locally, using Docker or Kubernetes.

## Docker

You can run containerbay with:

```bash
docker run -p 80:8080 \
           -e CONTAINERBAY_LISTENADDR=:8080 \
           -e CONTAINERBAY_MAGICDNS= \
           -e CONTAINERBAY_MAXSIZE=100MB \
           -e CONTAINERBAY_CLEANUPINTERVAL=1h \
           -ti --restart=always --name containerbay quay.io/mudler/containerbay run
```

The API endpoint will be accessible at `localhost:80`

## Kubernetes

There is a sample [deployment file](https://github.com/mudler/containerbay/blob/master/kube/deployment.yaml) in `kube`. Edit the ingress definition to fit your needs, there is also a traefik example configuration file for a wildcard DNS setup (magicDNS*)

## Locally

As containerbay is a static binary, you can just download the binary from the [releases](https://github.com/mudler/containerbay/releases) and run it locally.

Containerbay from the CLI can `run` as a proxy for multiple images, or run `standalone` to point only to a specific image/tag. 

There is also available a `pack` subcommand as utility to create docker-loadable images from folders and directories.

## Run standalone mode

Containerbay can also be used to serve a single container image reference only, for instance:

```bash
containerbay standalone <image/reference:tag>
```

Will start the API server serving the image on the default port (8080).

## Caveats

DockerHub applies pull rate limits to manifest fetching, `containerbay` could hit those limit depending on the service usage. Other container registries like e.g. `quay.io` don't have such limitations.

# Support containerbay.io

Currently containerbay is hosted merely on my own expenses, if you rely on this service, consider to donate or sponsor hosting for this service!

# Author

Ettore Di Giacinto

# Credits

Icons made by <a href="https://www.freepik.com" title="Freepik">Freepik</a> from <a href="https://www.flaticon.com/" title="Flaticon">www.flaticon.com</a>

# License

GPL-3
