package main

import (
	"context"
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
		"gocd-agent-scripts",
		"gocd-configuration",
		"auth-api-v2",
		"orion-service",
		"account-security-service",
		"infrastructure-deployment",
		"account-security-api-v1",
		"account-security-service-deployment",
		"ansible-host-ansible-role",
		"ansible-role-ntp-centos",
		"ansible-tower-bootstrap",
		"ansible-tower-install",
		"api-contract",
		"api-parent",
		"api-test",
		"apigee-ansible-modules",
		"atlas",
		"atlas-service-deployment-role",
		"atlas-service-base-docker-image",
		"auth-api-v1",
		"developer-osx",
		"docker-host-ansible-role",
		"coreplatform-vm",
		"environment-setup",
		"platform-workstream",
		"coreplatform",
		"platform-engineering-large-tests",
		"Platform-Engineering-Area-Handbook",
	}
)

func main() {
	token := "<<secret>>"
	githubUrl := "<<secret>>"

	oauthStaticToken := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})

	ctx := context.Background()
	oauthClient := oauth2.NewClient(ctx, oauthStaticToken)

	client := github.NewClient(oauthClient)
	url, _ := url.Parse(githubUrl)
	repoOwner := "Core-Platform"

	client.BaseURL = url

	file, _ := os.Create("result.csv")
	defer file.Close()

	writer := csv.NewWriter(os.Stdout)
	defer writer.Flush()

	for _, r := range repos {

		opt := &github.PullRequestListOptions{State: "closed", ListOptions: github.ListOptions{PerPage: 10}}

		var prs prArray

		_, resp, _ := client.PullRequests.List(ctx, repoOwner, r, opt)

		chPr := make(chan github.PullRequest)
		chFinished := make(chan bool)

		for i := 1; i <= resp.LastPage; i++ {
			go getPageUrl(ctx, client, repoOwner, r, chPr, chFinished, i)

		}

		for c := 0; c < resp.LastPage; {
			select {
			case pullReq := <-chPr:
				prs = append(prs, pr{Id: *pullReq.Number, Duration: pullReq.ClosedAt.Sub(*pullReq.CreatedAt), Date: pullReq.CreatedAt.Format("2006-01-02")})
				writer.Write([]string{
					r,
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

func getPageUrl(ctx context.Context, client *github.Client, owner, repo string, ch chan github.PullRequest, chFinished chan bool, page int) {
	opt := &github.PullRequestListOptions{State: "closed", ListOptions: github.ListOptions{PerPage: 10, Page: page}}

	pullRequests, _, err := client.PullRequests.List(ctx, owner, repo, opt)

	if err == nil {
		for _, pr := range pullRequests {
			ch <- *pr
		}
	}
	defer func() {
		chFinished <- true
	}()

}
