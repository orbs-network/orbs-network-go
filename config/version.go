package config

var SemanticVersion string
var CommitVersion string

type Version struct {
	Semantic string
	Commit   string
}

func GetVersion() Version {
	return Version{
		Semantic: SemanticVersion,
		Commit:   CommitVersion,
	}
}

func (v Version) String() string {
	return v.Semantic + "\n" + v.Commit
}
