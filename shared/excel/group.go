package excel

import "maps"

// expands a parent across its children, one row per child, with the parent's cells
// on the first row only. A childless parent still emits a row, so none disappears.
func Group(parent Row, children []Row) []Row {
	if len(children) == 0 {
		return []Row{parent}
	}

	rows := make([]Row, 0, len(children))
	for i, child := range children {
		if i > 0 {
			// Later rows carry only the child's cells; the parent's columns stay
			// blank because a key the row does not set renders as an empty cell.
			rows = append(rows, child)
			continue
		}
		rows = append(rows, merge(parent, child))
	}
	return rows
}

// combines two rows without mutating either, since a caller may reuse the parent
func merge(parent, child Row) Row {
	row := make(Row, len(parent)+len(child))
	maps.Copy(row, parent)
	maps.Copy(row, child)
	return row
}
