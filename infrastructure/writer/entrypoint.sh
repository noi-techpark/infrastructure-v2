#!/bin/sh

mkdir -p ~/.m2
cat > ~/.m2/settings.xml << EOF
<settings>
    <localRepository>$PWD/.m2</localRepository>
</settings>
EOF

$@
