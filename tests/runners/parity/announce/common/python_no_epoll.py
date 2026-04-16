#!/usr/bin/env python3
import runpy
import sys

import RNS.vendor.platformutils as platformutils

platformutils.use_epoll = lambda: False

target = sys.argv[1]
sys.argv = sys.argv[1:]
runpy.run_path(target, run_name="__main__")
