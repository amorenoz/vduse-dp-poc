#!/bin/sh

set -e

usage()
{
    /bin/echo -e "This is an entrypoint script for VDUSE Device Plugin"
    /bin/echo -e ""
    /bin/echo -e "./entrypoint.sh"
}

exec vduse-dp $@
