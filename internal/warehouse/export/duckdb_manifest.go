package export

import (
	"fmt"
	"strings"
)

func BuildManifestSQL(viewName string, files []string) string {
	quoted := make([]string, 0, len(files))
	for _, file := range files {
		quoted = append(quoted, fmt.Sprintf("'%s'", strings.ReplaceAll(file, "'", "''")))
	}
	return fmt.Sprintf("CREATE OR REPLACE VIEW %s AS\nSELECT * FROM read_parquet([%s]);\n", viewName, strings.Join(quoted, ", "))
}
