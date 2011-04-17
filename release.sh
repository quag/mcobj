#!/bin/bash

7z.exe a mcobj-$(git describe)-windows.7z mcobj.exe blocks.json
