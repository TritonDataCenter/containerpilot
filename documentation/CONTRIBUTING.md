# Contributing and developing ContainerPilot documentation

The official ContainerPilot documentation is published at [joyent.com/containerpilot/docs](https://www.joyent.com/containerpilot/docs), but that content is open source and maintained in this repo.

### Formatting

Documentation is published using [Kirby](https://getkirby.com), a mildly quirky file-based CMS. One of Kirby's quirks is that every "page" on the website is actually a directory in the repo, and each of those directories is prefixed with a number to order the pages in the navigation. The content of that page is in the `README.md` file in that directory, and any images should be added to the directory for the page they appear in.

[Kirby supports Markdown](https://getkirby.com/docs/content/text), and where possible, all content should implement [Github flavored Markdown syntax](https://help.github.com/articles/basic-writing-and-formatting-syntax/). The exception is when Kirby's syntax parser doesn't support Github's flavor.

### Previewing

A Dockerfile is included to make it possible to preview the content of the docs when editing them.

```bash
docker build --no-cache -t mydocs
docker run -d -p 80:80 --name mydocs mydocs
```

`--no-cache` is suggested because changes in the theme repo won't be fetched if the layer is cached.
