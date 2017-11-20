package envelopes

type Effect struct {
	impacts map[string]int64
}

func (e Effect) deepCopy() (result Effect) {
	result.impacts = make(map[string]int64, len(e.impacts))

	for budg, impact := range e.impacts {
		result.impacts[budg] = impact
	}

	return
}
