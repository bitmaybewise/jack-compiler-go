package writer

import (
	"encoding/xml"
	"strings"
)

func Output(out *strings.Builder, value any) error {
	result, err := xml.MarshalIndent(value, "", " ")
	if err != nil {
		return err
	}

	out.Write(result)
	return nil
}
