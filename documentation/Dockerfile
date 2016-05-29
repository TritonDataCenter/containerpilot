FROM joyent/kirby-preview-base:latest

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
