from distutils.core import setup
import py2exe

from lapera import *

setup(console=['./lapera/__main__.py'])