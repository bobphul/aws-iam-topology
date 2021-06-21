package main

import (
    "context"
    "fmt"
    "sync"
    "regexp"
    "runtime"

    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/iam"

    "github.com/neo4j/neo4j-go-driver/neo4j"
)

func check(err error) {
    if err != nil {
        panic(err)
    }
}

func getUserData(details *iam.GetAccountAuthorizationDetailsOutput, wg *sync.WaitGroup) {
    defer wg.Done()

    driver, err := connectGraph()
    check(err)

    defer driver.Close()

    for _, user := range details.UserDetailList {
        _, err := addUserData(driver, *user.UserName, *user.Arn, *user.UserId)
        check(err)
        for _, user_attached_group := range user.GroupList{
	    _, err = addUserGroupRelationData(driver, *user.UserName, user_attached_group)
	    check(err)
        }
        for _, user_attached_policy := range user.AttachedManagedPolicies{
	    _, err := addPolicyData(driver, *user_attached_policy.PolicyName)
	    check(err)
	    _, err = addUserPolicyRelationData(driver, *user_attached_policy.PolicyName, *user.UserName)
	    check(err)
        }
    }
}

func getGroupData(details *iam.GetAccountAuthorizationDetailsOutput, wg *sync.WaitGroup) {
    //defer wg.Done()

    driver, err := connectGraph()
    check(err)

    defer driver.Close()

    for _, group := range details.GroupDetailList {
	_, err := addGroupData(driver, *group.GroupName, *group.Arn, *group.GroupId)
	check(err)
	for _, group_attached_policy := range group.AttachedManagedPolicies{
	    _, err := addPolicyData(driver, *group_attached_policy.PolicyName)
	    check(err)
	    _, err = addGroupPolicyRelationData(driver, *group_attached_policy.PolicyName, *group.GroupName)
	    check(err)
	}
    }
}

func getRoleData(details *iam.GetAccountAuthorizationDetailsOutput, wg *sync.WaitGroup) {
    defer wg.Done()

    driver, err := connectGraph()
    check(err)

    defer driver.Close()

    for _, role := range details.RoleDetailList {
	for _, role_attached_policy := range role.AttachedManagedPolicies{
            match, _ := regexp.MatchString("AWSServiceRoleFor", *role.RoleName)
            if match == false {
	        _, err := addPolicyData(driver, *role_attached_policy.PolicyName)
	        check(err)
                fmt.Println("ROLE:", *role.RoleName, "--> POLICY:", *role_attached_policy.PolicyName)
	    }
	}
    }
}

func connectGraph() (neo4j.Driver, error) {
    dbUri := "neo4j://localhost:7687"
    driver, err := neo4j.NewDriver(dbUri, neo4j.BasicAuth("neo4j", "3joh22a", ""))
    check(err)
    return driver, nil
}

func addUserData(driver neo4j.Driver, name string, arn string, id string) (int64, error) {
    session := driver.NewSession(neo4j.SessionConfig{})
    defer session.Close()
    var uid interface{}
    var err error
    uid, err = session.ReadTransaction(matchUserNodeTxFunc(name, arn, id))
    check(err)

    if uid != nil {return uid.(int64), nil}

    _, err = session.WriteTransaction(addUserNodeTxFunc(name, arn, id))
    check(err)

    uid, err = session.ReadTransaction(matchUserNodeTxFunc(name, arn, id))
    check(err)

    return uid.(int64), nil
}

func addUserNodeTxFunc(name string, arn string, id string) neo4j.TransactionWork {
    return func(tx neo4j.Transaction) (interface{}, error) {
        result, err := tx.Run("CREATE (n:User {name: $name, arn: $arn, id: $id})", map[string]interface{}{
            "name": name,
	    "arn": arn,
	    "id": id,
        })
        if err != nil {return nil, err}
        return result.Consume()
    }
}

func matchUserNodeTxFunc(name string, arn string, id string) neo4j.TransactionWork {
    return func(tx neo4j.Transaction) (interface{}, error) {
        result, err := tx.Run("MATCH (n:User {name: $name, arn: $arn, id: $id}) RETURN id(n)", map[string]interface{}{
           "name": name,
	   "arn": arn,
	   "id": id,
        })
	check(err)

	if result.Next() {
            return result.Record().Values[0], nil
        }

	return nil, nil
    }
}

func addGroupData(driver neo4j.Driver, name string, arn string, id string) (int64, error) {
    session := driver.NewSession(neo4j.SessionConfig{})
    defer session.Close()
    var gid interface{}
    var err error
    gid, err = session.ReadTransaction(matchGroupNodeTxFunc(name, arn, id))
    check(err)

    if gid != nil {return gid.(int64), nil}

    _, err = session.WriteTransaction(addGroupNodeTxFunc(name, arn, id))
    check(err)

    gid, err = session.ReadTransaction(matchGroupNodeTxFunc(name, arn, id))
    check(err)

    return gid.(int64), nil
}

func addGroupNodeTxFunc(name string, arn string, id string) neo4j.TransactionWork {
    return func(tx neo4j.Transaction) (interface{}, error) {
        result, err := tx.Run("CREATE (g:Group {name: $name, arn: $arn, id: $id})", map[string]interface{}{
            "name": name,
            "arn": arn,
            "id": id,
        })
        if err != nil {return nil, err}
        return result.Consume()
    }
}

func matchGroupNodeTxFunc(name string, arn string, id string) neo4j.TransactionWork {
    return func(tx neo4j.Transaction) (interface{}, error) {
        result, err := tx.Run("MATCH (g:Group {name: $name, arn: $arn, id: $id}) RETURN id(g)", map[string]interface{}{
           "name": name,
           "arn": arn,
           "id": id,
        })
        check(err)

        if result.Next() {
            return result.Record().Values[0], nil
        }

        return nil, nil
    }
}

func addPolicyData(driver neo4j.Driver, name string) (int64, error) {
    session := driver.NewSession(neo4j.SessionConfig{})
    defer session.Close()
    var pid interface{}
    var err error
    pid, err = session.ReadTransaction(matchPolicyNodeTxFunc(name))
    check(err)

    if pid != nil {return pid.(int64), nil}

    _, err = session.WriteTransaction(addPolicyNodeTxFunc(name))
    check(err)

    pid, err = session.ReadTransaction(matchPolicyNodeTxFunc(name))
    check(err)

    return pid.(int64), nil
}

func addPolicyNodeTxFunc(name string) neo4j.TransactionWork {
    return func(tx neo4j.Transaction) (interface{}, error) {
        result, err := tx.Run("CREATE (p:Policy {name: $name})", map[string]interface{}{
            "name": name,
        })
        if err != nil {return nil, err}
        return result.Consume()
    }
}

func matchPolicyNodeTxFunc(name string) neo4j.TransactionWork {
    return func(tx neo4j.Transaction) (interface{}, error) {
        result, err := tx.Run("MATCH (p:Policy {name: $name}) RETURN id(p)", map[string]interface{}{
           "name": name,
        })
	check(err)

	if result.Next() {
            return result.Record().Values[0], nil
        }

	return nil, nil
    }
}

func addUserPolicyRelationData(driver neo4j.Driver, sourceName string, targetName string) (int64, error){
    session := driver.NewSession(neo4j.SessionConfig{})
    defer session.Close()
    var rid interface{}
    var err error
    rid, err = session.ReadTransaction(matchUserPolicyRelationTxFunc(sourceName, targetName))
    check(err)

    if rid != nil {
        return rid.(int64), nil
    }

    _, err = session.WriteTransaction(addUserPolicyRelationTxFunc(sourceName, targetName))
    check(err)

    rid, err = session.ReadTransaction(matchUserPolicyRelationTxFunc(sourceName, targetName))
    check(err)

    if rid != nil {
        return rid.(int64), nil
    }

    return -1, nil
}

func addGroupPolicyRelationData(driver neo4j.Driver, sourceName string, targetName string) (int64, error){
    session := driver.NewSession(neo4j.SessionConfig{})
    defer session.Close()
    var rid interface{}
    var err error
    rid, err = session.ReadTransaction(matchGroupPolicyRelationTxFunc(sourceName, targetName))
    check(err)

    if rid != nil {
        return rid.(int64), nil
    }

    _, err = session.WriteTransaction(addGroupPolicyRelationTxFunc(sourceName, targetName))
    check(err)

    rid, err = session.ReadTransaction(matchGroupPolicyRelationTxFunc(sourceName, targetName))
    check(err)

    if rid != nil {
        return rid.(int64), nil
    }

    return -1, nil
}

func addUserGroupRelationData(driver neo4j.Driver, sourceName string, targetName string) (int64, error){
    session := driver.NewSession(neo4j.SessionConfig{})
    defer session.Close()
    var rid interface{}
    var err error
    rid, err = session.ReadTransaction(matchUserGroupRelationTxFunc(sourceName, targetName))
    check(err)

    if rid != nil {
        return rid.(int64), nil
    }

    _, err = session.WriteTransaction(addUserGroupRelationTxFunc(sourceName, targetName))
    check(err)

    rid, err = session.ReadTransaction(matchUserGroupRelationTxFunc(sourceName, targetName))
    check(err)

    if rid != nil {
        return rid.(int64), nil
    }

    return -1, nil
}

func addUserPolicyRelationTxFunc(sourceName string, targetName string) neo4j.TransactionWork {
    return func(tx neo4j.Transaction) (interface{}, error) {
        result, err := tx.Run("MATCH (u:User),(p:Policy) WHERE p.name=$sName AND u.name=$tName CREATE (p)-[rel:ATTACHED {relation: p.name+'-->'+u.name}]->(u)", map[string]interface{}{
            "sName": sourceName,
            "tName": targetName,
        })
        if err != nil {return nil, err}

	if result.Next() {
            return result.Record().Values[0], nil
        }

        return nil, nil
    }
}

func addGroupPolicyRelationTxFunc(sourceName string, targetName string) neo4j.TransactionWork {
    return func(tx neo4j.Transaction) (interface{}, error) {
        result, err := tx.Run("MATCH (g:Group),(p:Policy) WHERE p.name=$sName AND g.name=$tName CREATE (p)-[rel:ATTACHED {relation: p.name+'-->'+g.name}]->(g)", map[string]interface{}{
            "sName": sourceName,
            "tName": targetName,
        })
        if err != nil {return nil, err}

	if result.Next() {
            return result.Record().Values[0], nil
        }

        return nil, nil
    }
}

func addUserGroupRelationTxFunc(sourceName string, targetName string) neo4j.TransactionWork {
    return func(tx neo4j.Transaction) (interface{}, error) {
        result, err := tx.Run("MATCH (u:User),(g:Group) WHERE u.name=$sName AND g.name=$tName CREATE (u)-[rel:MEMEBER_OF {relation: u.name+'-->'+g.name}]->(g)", map[string]interface{}{
            "sName": sourceName,
            "tName": targetName,
        })
        if err != nil {return nil, err}

        if result.Next() {
            return result.Record().Values[0], nil
        }

        return nil, nil
    }
}

func matchUserPolicyRelationTxFunc(sourceName string, targetName string) neo4j.TransactionWork {
    return func(tx neo4j.Transaction) (interface{}, error) {
        result, err := tx.Run("MATCH (p:Policy {name: $sName})-[rel:ATTACHED]->(u:User {name: $tName}) RETURN id(rel)", map[string]interface{}{
            "sName": sourceName,
            "tName": targetName,
        })
        if err != nil {
            return nil, err
        }

        if result.Next() {
            return result.Record().Values[0], nil
        }

        return nil, nil
    }
}

func matchGroupPolicyRelationTxFunc(sourceName string, targetName string) neo4j.TransactionWork {
    return func(tx neo4j.Transaction) (interface{}, error) {
        result, err := tx.Run("MATCH (p:Policy {name: $sName})-[rel:ATTACHED]->(g:Group {name: $tName}) RETURN id(rel)", map[string]interface{}{
            "sName": sourceName,
            "tName": targetName,
        })
        if err != nil {
            return nil, err
        }

        if result.Next() {
            return result.Record().Values[0], nil
        }

        return nil, nil
    }
}

func matchUserGroupRelationTxFunc(sourceName string, targetName string) neo4j.TransactionWork {
    return func(tx neo4j.Transaction) (interface{}, error) {
        result, err := tx.Run("MATCH (u)-[rel]->(g) WHERE u.name=$sName AND g.name=$tName RETURN id(rel)", map[string]interface{}{
            "sName": sourceName,
            "tName": targetName,
        })
        if err != nil {return nil, err}

        if result.Next() {
            return result.Record().Values[0], nil
        }

        return nil, nil
    }
}

func main() {
    runtime.GOMAXPROCS(runtime.NumCPU())

    cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithSharedConfigProfile("master"),)
    check(err)

    client := iam.NewFromConfig(cfg)

    iam_details, err := client.GetAccountAuthorizationDetails(context.TODO(), &iam.GetAccountAuthorizationDetailsInput{})
    check(err)

    var wg sync.WaitGroup

    wg.Add(2)

    getGroupData(iam_details, &wg)
    go getRoleData(iam_details, &wg)
    go getUserData(iam_details, &wg)

    wg.Wait()
}

