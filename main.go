package main

import (
	"golang.org/x/net/context"
	"encoding/csv"
	"fmt"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	"net/url"
	"os"
	"sort"
	"time"
)

type pr struct {
	Id       int
	Duration time.Duration
	Date     string
}

type prArray []pr

func (s prArray) Len() int {
	return len(s)
}
func (s prArray) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s prArray) Less(i, j int) bool {
	return s[i].Id < s[j].Id
}

var (
	repos = []string{
		"<<secret>>",
		"<<secret>>",
	}
)

func main() {
	token := "<<secret>>"
	githubUrl := "<<secret>>"
	repoOwner := "<<secret>>"

	oauthStaticToken := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})

	ctx := context.Background()
	oauthClient := oauth2.NewClient(ctx, oauthStaticToken)

	client := github.NewClient(oauthClient)
	githubBaseUrl, _ := url.Parse(githubUrl)

	client.BaseURL = githubBaseUrl

	writer := csv.NewWriter(os.Stdout)
	defer writer.Flush()

	for _, repoName := range repos {

		pullRequestListOptions := &github.PullRequestListOptions{State: "closed", ListOptions: github.ListOptions{}}

		var prs prArray

		_, resp, _ := client.PullRequests.List(ctx, repoOwner, repoName, pullRequestListOptions)

		channelPullRequest := make(chan github.PullRequest)
		chFinished := make(chan bool)

		for prPageNumber := 1; prPageNumber <= resp.LastPage; prPageNumber++ {
			go getPageUrl(ctx, client, repoOwner, repoName, channelPullRequest, chFinished, prPageNumber)
		}

		for c := 0; c < resp.LastPage; {
			select {
			case pullReq := <-channelPullRequest:
				prs = append(prs, pr{
					Id:       *pullReq.Number,
					Duration: pullReq.ClosedAt.Sub(*pullReq.CreatedAt),
					Date:     pullReq.CreatedAt.Format("2006-01-02"),
				})
				writer.Write([]string{
					repoName,
					pullReq.CreatedAt.Format("2006-01-02"),
					fmt.Sprintf("%f", pullReq.ClosedAt.Sub(*pullReq.CreatedAt).Hours()),
					*pullReq.User.Login,
				})
			case <-chFinished:
				c++
			}
		}

		sort.Sort(prs)

	}

}

func getPageUrl(ctx context.Context, client *github.Client, repoOwner string, repoName string, channelPullRequest chan github.PullRequest, chFinished chan bool, pullRequestPageNumber int) {
	opt := &github.PullRequestListOptions{
		State: "closed",
		ListOptions: github.ListOptions{
			PerPage: 10,
			Page:    pullRequestPageNumber,
		},
	}

	pullRequests, _, err := client.PullRequests.List(ctx, repoOwner, repoName, opt)

	if err == nil {
		for _, pr := range pullRequests {
			channelPullRequest <- *pr
		}
	}
	defer func() {
		chFinished <- true
	}()

}
