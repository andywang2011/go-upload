package upload

type errNotFoundLike struct{ msg string }

func (e errNotFoundLike) Error() string { return e.msg }
