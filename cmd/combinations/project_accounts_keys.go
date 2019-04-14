package combinations

import (
	"fmt"
	"github.com/codegangsta/cli"
	"github.com/fionera/TeamdriveManager/api"
	"github.com/fionera/TeamdriveManager/api/cloudresourcemanager"
	"github.com/fionera/TeamdriveManager/api/iam"
	"github.com/fionera/TeamdriveManager/api/servicemanagement"
	. "github.com/fionera/TeamdriveManager/config"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"sync"
)

func NewProjectAccountsKeysCommand() cli.Command {
	return cli.Command{
		Name:   "project_accounts_keys",
		Usage:  "Create a Project, fill it with 100 Accounts and create the Keys for it",
		Action: CmdCreateProjectAccountsKeys,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name: "project-id",
			},
			cli.StringFlag{
				Name: "organization",
			},
		},
	}
}

func CmdCreateProjectAccountsKeys(c *cli.Context) {
	projectId := c.String("project-id")
	organization := c.String("organization")

	if projectId == "" {
		logrus.Error("Please supply the ProjectID to use")
		return
	}

	if organization == "" {
		if App.AppConfig.Organization != "" {
			organization = App.AppConfig.Organization
		} else {
			logrus.Error("Please supply the Organization to use")
			return
		}
	}

	client, err := api.CreateClient(App.AppConfig.ServiceAccountFile, App.AppConfig.Impersonate, []string{cloudresourcemanager.CloudPlatformScope, servicemanagement.ServiceManagementScope})
	if err != nil {
		logrus.Panic(err)
		return
	}

	crmApi, err := cloudresourcemanager.NewApi(client)
	if err != nil {
		logrus.Panic(err)
		return
	}

	smApi, err := servicemanagement.NewApi(client)
	if err != nil {
		logrus.Panic(err)
		return
	}

	iamApi, err := iam.NewApi(client)
	if err != nil {
		logrus.Panic(err)
		return
	}

	err = crmApi.CreateProject(projectId, organization)
	if err != nil {
		logrus.Panic(err)
		return
	}

	err = smApi.EnableApi("project:"+projectId, servicemanagement.DriveApi)
	if err != nil {
		logrus.Panic(err)
		return
	}

	var requests sync.WaitGroup
	var running int
	for i := 1; i <= 100; i++ {
		requests.Add(1)
		running++

		go func(i int) {
			defer requests.Done()

			accountId := fmt.Sprintf("service-account-%d", i)

		createServiceAccount:
			logrus.Infof("Creating Service Account: %s", accountId)
			serviceAccount, err := iamApi.CreateServiceAccount(projectId, accountId, "")
			if err != nil {
				logrus.Panic(err)
				goto createServiceAccount
			}

		createApiKey:
			logrus.Infof("Creating Key for Account: %s", accountId)
			serviceAccountKey, err := iamApi.CreateServiceAccountKey(serviceAccount)
			if err != nil {
				logrus.Panic(err)
				goto createApiKey
			}

			json, err := serviceAccountKey.MarshalJSON()
			if err != nil {
				logrus.Panic(err)
				return
			}

			err = ioutil.WriteFile(serviceAccount.ProjectId+"_"+serviceAccount.DisplayName+".json", json, 0755)
			if err != nil {
				logrus.Panic(err)
				return
			}
		}(i)

		if running > App.Flags.Concurrency {
			requests.Wait()
			running = 0
		}
	}

	requests.Wait()
	logrus.Infof("Done :3")
}
