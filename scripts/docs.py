#!/usr/bin/env python
from __future__ import print_function
import os
import re
import shutil


# rewrites the title as Kirby front-matter
def add_front_matter(content):
    title = content.split('\n', 1)[0].replace('# ', '')
    rest = content.split('\n')[1:]
    front_matter = """Title: {}

----

Text:
""".format(title)
    content = [front_matter] + rest
    content = '\n'.join(content)
    return content


# rewrites all markdown links to indexes (in-place)
def rewrite_links(content):

    def rewrite_markdown_link(matchobj):
        match = matchobj.group(0)
        match = match.replace('.md', '')
        match = re.sub(r'[0-9]{2}\-', '', match)
        return match

    content = re.sub(
        r'\(\./.*?\.md.*?\)',
        rewrite_markdown_link,
        content)

    def rewrite_json_link(matchobj):
        match = matchobj.group(0)
        match = match.replace(
            './examples',
            '/containerpilot/30-configuration/39-examples')
        return match

    content = re.sub(
        r'\(\./examples/.*?\.json5\)',
        rewrite_json_link,
        content)

    return content


# copy markdown files to the file structure that Kirby expects
def copy_markdown():
    os.mkdir('./build/docs')
    for dirpath, dirname, fnames in os.walk('docs'):
        for fname in fnames:
            if fname.endswith('.md') and fname != "README.md":

                source = '{}/{}'.format(dirpath, fname)
                dirname = fname.replace('.md', '')
                build_dir = './build/{}/{}'.format(dirpath, dirname)
                dest = '{}/docs.md'.format(build_dir)

                os.makedirs(build_dir)
                content = ''
                with open(source, 'r') as fr:
                    content = fr.read()

                content = add_front_matter(content)
                content = rewrite_links(content)

                with open(dest, 'w') as fw:
                    fw.write(content)

                print('{} -> {}'.format(source, dest))

# top-level index is weird exception to the structure
def fix_top_level():
    build_dir = 'build/docs/00-index'
    os.makedirs(build_dir)
    source = 'docs/README.md'
    content = ''
    with open(source, 'r') as fr:
        content = fr.read()

    content = add_front_matter(content)
    content = rewrite_links(content)

    dest = '{}/docs.md'.format(build_dir)
    with open(dest, 'w') as fw:
        fw.write(content)

    print('{} -> {}'.format(source, dest))


# configuration examples in JSON5 format
def copy_json_examples():
    dest_dir = 'build/docs/30-configuration/examples'
    os.makedirs(dest_dir)
    for dirpath, _, fnames in os.walk('docs/30-configuration/examples'):
        for fname in fnames:
            source = '{}/{}'.format(dirpath, fname)
            dest = '{}/{}'.format(dest_dir, fname)
            shutil.copy(source, dest)
            print('{} -> {}'.format(source, dest))


if __name__ == '__main__':
    copy_markdown()
    fix_top_level()
    copy_json_examples()
