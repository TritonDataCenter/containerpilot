# Contributing and developing ContainerPilot documentation

The official ContainerPilot documentation is published at [joyent.com/containerpilot/docs](https://www.joyent.com/containerpilot/docs), but that content is open source and maintained in this repo.

### Kirby

Documentation is published using [Kirby](https://getkirby.com), a mildly quirky file-based CMS. One of Kirby's quirks is that every "page" on the website is actually a directory in the repo, and each of those directories is prefixed with a number to order the pages in the navigation. The content of that page is in the `README.md` file in that directory, and any images should be added to the directory for the page they appear in.

[Kirby supports Markdown](https://getkirby.com/docs/content/text), and where possible, all content should implement [Github flavored Markdown syntax](https://help.github.com/articles/basic-writing-and-formatting-syntax/). The exception is when Kirby's syntax parser doesn't support Github's flavor.

### Previewing

A Dockerfile is included to make it possible to preview the content of the docs when editing them.

```bash
docker build -t mydocs
docker run -d -p 80:80 --name mydocs mydocs
```

The Dockerfile uses https://github.com/joyent/kirby-preview-base ([in Docker Hub](https://hub.docker.com/r/joyent/kirby-preview-base/)) for the base Kirby install and theme.

### Language styleguide

- Please do not force wrap text in `*.md` files
- Please make sure your links work in the Kirby preview; even if that results in links that don't work when rendered on Github.com
- Please use [serial, or Oxford commas](https://en.wikipedia.org/wiki/Serial_comma)
- Please be human, and respect the humanity of others