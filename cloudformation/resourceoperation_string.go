// Code generated by "stringer -type ResourceOperation -trimprefix Resource"; DO NOT EDIT.

package cloudformation

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[ResourceCreate-0]
	_ = x[ResourceUpdate-1]
	_ = x[ResourceDelete-2]
	_ = x[ResourceImport-3]
}

const _ResourceOperation_name = "CreateUpdateDeleteImport"

var _ResourceOperation_index = [...]uint8{0, 6, 12, 18, 24}

func (i ResourceOperation) String() string {
	if i < 0 || i >= ResourceOperation(len(_ResourceOperation_index)-1) {
		return "ResourceOperation(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _ResourceOperation_name[_ResourceOperation_index[i]:_ResourceOperation_index[i+1]]
}
