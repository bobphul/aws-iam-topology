package main

import (
    "context"
    "fmt"
    "log"
    "sync"
    "regexp"
    "runtime"

    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/iam"
)

func check(text string, err error) {
    if err != nil {
        log.Fatalf("Got an error retrieving : %s, %v", text, err)
	return
    }
}

func getUserData(details *iam.GetAccountAuthorizationDetailsOutput, wg *sync.WaitGroup) {
    defer wg.Done()
    for _, user := range details.UserDetailList {
        for _, user_attached_group := range user.GroupList{
            fmt.Println("USER:", *user.UserName, "--> GROUP:", user_attached_group)
        }
        for _, user_attached_policy := range user.AttachedManagedPolicies{
            fmt.Println("USER:", *user.UserName, "--> POLICY:", *user_attached_policy.PolicyName)
        }
    }
}

func getGroupData(details *iam.GetAccountAuthorizationDetailsOutput, wg *sync.WaitGroup) {
    defer wg.Done()
    for _, group := range details.GroupDetailList {
	for _, group_attached_policy := range group.AttachedManagedPolicies{
            fmt.Println("GROUP:", *group.GroupName, "--> POLICY:", *group_attached_policy.PolicyName)
	}
    }
}

func getRoleData(details *iam.GetAccountAuthorizationDetailsOutput, wg *sync.WaitGroup) {
    defer wg.Done()
    for _, role := range details.RoleDetailList {
	for _, role_attached_policy := range role.AttachedManagedPolicies{
            match, _ := regexp.MatchString("AWSServiceRoleFor", *role.RoleName)
            if match == false {
                fmt.Println("ROLE:", *role.RoleName, "--> POLICY:", *role_attached_policy.PolicyName)
	    }
	}
    }
}

func main() {
    runtime.GOMAXPROCS(runtime.NumCPU())

    cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithSharedConfigProfile("master"),)
    if err != nil {
        log.Fatalf("unable to load SDK config, %v", err)
    }

    client := iam.NewFromConfig(cfg)

    iam_details, err := client.GetAccountAuthorizationDetails(context.TODO(), &iam.GetAccountAuthorizationDetailsInput{})
    check("account", err)

    var wg sync.WaitGroup

    wg.Add(3)

    go getUserData(iam_details, &wg)
    go getGroupData(iam_details, &wg)
    go getRoleData(iam_details, &wg)

    wg.Wait()
}

