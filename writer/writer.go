package writer

import (
	"encoding/xml"
	"fmt"
	"strings"
)

func Output(out *strings.Builder, value any) error {
	result, err := xml.MarshalIndent(value, "", " ")
	if err != nil {
		return err
	}

	fmt.Printf("%s\n", result)
	out.Write(result)

	return nil
}
