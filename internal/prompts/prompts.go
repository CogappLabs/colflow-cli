package prompts

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/lukew-cogapp/colflow-cli/internal/client"
	"github.com/lukew-cogapp/colflow-cli/internal/format"
)

type Item[T any] struct {
	Display string
	Value   T
}

func Pick[T any](label string, items []Item[T]) (T, bool) {
	var zero T
	if len(items) == 0 {
		return zero, false
	}

	fmt.Printf("\n%s:\n", label)
	fmt.Println("  0) Cancel")
	for i, item := range items {
		fmt.Printf("  %d) %s\n", i+1, item.Display)
	}

	fmt.Print("\nEnter number: ")
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return zero, false
	}
	line = strings.TrimSpace(line)
	idx, err := strconv.Atoi(line)
	if err != nil || idx < 1 || idx > len(items) {
		return zero, false
	}
	return items[idx-1].Value, true
}

func SelectRun() (string, bool) {
	runs, err := client.GetRuns(10, "")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return "", false
	}
	items := make([]Item[string], len(runs))
	for i, r := range runs {
		short := r.RunID
		if len(short) > 8 {
			short = short[:8]
		}
		items[i] = Item[string]{
			Display: fmt.Sprintf("%s  %s  %s  %s", short, format.ColorStatus(r.Status), r.JobName, format.TimeAgo(r.StartTime)),
			Value:   r.RunID,
		}
	}
	return Pick("Recent runs", items)
}

func SelectAsset() (string, bool) {
	assets, err := client.GetAssets()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return "", false
	}
	items := make([]Item[string], len(assets))
	for i, a := range assets {
		group := "ungrouped"
		if a.GroupName != nil {
			group = *a.GroupName
		}
		path := strings.Join(a.AssetKey.Path, "/")
		items[i] = Item[string]{
			Display: fmt.Sprintf("%s (%s)", path, group),
			Value:   path,
		}
	}
	return Pick("Assets", items)
}

func SelectJob() (string, bool) {
	jobs, err := client.GetJobs()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return "", false
	}
	user := []client.Job{}
	for _, j := range jobs {
		if !strings.HasPrefix(j.Name, "__") {
			user = append(user, j)
		}
	}
	if len(user) == 1 {
		return user[0].Name, true
	}
	items := make([]Item[string], len(user))
	for i, j := range user {
		display := j.Name
		if j.Description != nil && *j.Description != "" {
			display = fmt.Sprintf("%s — %s", j.Name, *j.Description)
		}
		items[i] = Item[string]{Display: display, Value: j.Name}
	}
	return Pick("Jobs", items)
}
