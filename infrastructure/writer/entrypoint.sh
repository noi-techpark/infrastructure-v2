#!/usr/bin/env bash

# SPDX-FileCopyrightText: NOI Techpark <digital@noi.bz.it>
#
# SPDX-License-Identifier: CC0-1.0

mkdir -p ~/.m2
cat > ~/.m2/settings.xml << EOF
<settings>
    <localRepository>$PWD/.m2</localRepository>
</settings>
EOF

$@
