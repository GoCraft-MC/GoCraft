package permission

import (
	"encoding/json"
	"fmt"
)

func Validate(document Document) error {
	return validateDocument(document)
}

func Clone(document Document) Document {
	return cloneDocument(document)
}

func cloneDocument(document Document) Document {
	data, _ := json.Marshal(document)
	var clone Document
	_ = json.Unmarshal(data, &clone)
	return clone
}

func validateDocument(document Document) error {
	if err := document.Validate(); err != nil {
		return err
	}
	visiting, visited := map[string]bool{}, map[string]bool{}
	var visit func(string) error
	visit = func(name string) error {
		if visiting[name] {
			return fmt.Errorf("permission group inheritance cycle at %q", name)
		}
		if visited[name] {
			return nil
		}
		visiting[name] = true
		for _, parent := range document.Groups[name].Parents {
			if err := visit(Normalize(parent)); err != nil {
				return err
			}
		}
		visiting[name], visited[name] = false, true
		return nil
	}
	for name := range document.Groups {
		if err := visit(name); err != nil {
			return err
		}
	}
	return nil
}
