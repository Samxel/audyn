package handler

import (
	"fmt"
	"net/http"
	"strings"
)

func ServeNZB(w http.ResponseWriter, r *http.Request) {
	// extract album id
	id := strings.TrimPrefix(r.URL.Path, "/download/")

	w.Header().Set("Content-Type", "application/x-nzb")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.nzb"`, id))

	fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE nzb PUBLIC "-//newzBin//DTD NZB 1.1//EN" "http://www.newzbin.com/DTD/nzb/nzb-1.1.dtd">
<nzb xmlns="http://www.newzbin.com/DTD/2003/nzb">
  <head>
    <meta type="category">music</meta>
  </head>
  <file subject="audyn-deezer-%s">
    <groups><group>alt.binaries.music</group></groups>
    <segments>
      <segment bytes="150000000" number="1">audyn-deezer-%s@audyn</segment>
    </segments>
  </file>
</nzb>`, id, id)
}
