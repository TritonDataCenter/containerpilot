FROM php:5.6-apache

# Allow rewrites in Apache httpd
RUN a2enmod rewrite

# Get some packages, especially Git, but also some convenience items
RUN apt-get update && apt-get install -y --no-install-recommends \
        curl \
        git \
        less \
        vim \
    && rm -rf /var/lib/apt/lists/*

# Add Kirby base
RUN mkdir -p /var/www/html/ \
    && git clone --recursive --single-branch \
        -b working \
        https://github.com/misterbisson/docs.git \
        /var/www/html

# Add our content
# This is a bit sloppy, but it makes the repo directory structure flatter
# ...and avoids being specific about the pages/directories to add to the container
COPY . /var/www/html/content/1-containerpilot/1-docs/

# Move and clean up some items added above
RUN cd /var/www/html/content/1-containerpilot/1-docs/ \
    && rm -f \
        .dockerignore \
        Dockerfile \
        CONTRIBUTING.md \
    && mv site.md /var/www/html/content/.