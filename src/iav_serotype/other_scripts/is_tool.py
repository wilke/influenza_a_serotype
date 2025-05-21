#!/usr/bin/env python

def is_tool(name):
    """Check whether `name` is on PATH."""
    from shutil import which as find_executable
    return find_executable(name) is not None