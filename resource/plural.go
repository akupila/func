package resource

func plural(count int, one, many string) string {
	if count == 1 {
		return one
	}
	return many
}
