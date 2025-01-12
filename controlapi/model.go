package controlapi

type Actor struct {
	Id       string  `json:"id"`
	Title    string  `json:"title"`
	Ref      *string `json:"ref"`
	Color    *string `json:"color"`
	Picture  *string `json:"picture"`
	FormId   int64   `json:"formId"`
	Position struct {
		X float64 `json:"x"`
		Y float64 `json:"y"`
	} `json:"position"`
}

type Edge struct {
	Id     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
}
type LayerActors struct {
	Nodes []Actor `json:"nodes"`
	Edges []Edge  `json:"edges"`
}

func (l LayerActors) Actor(id string) Actor {
	for _, a := range l.Nodes {
		if id == a.Id {
			return a
		}
	}
	panic("actor not found")
}
