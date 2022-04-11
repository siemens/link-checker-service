 #!/bin/bash
 set -euo pipefail
 IFS=$'\n\t'

 python3 -m http.server 8092
