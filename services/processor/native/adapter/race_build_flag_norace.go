//+build race

package adapter

func addRaceFlagIfNeeded(args []string) []string {
	return append(args, "-race")
}
