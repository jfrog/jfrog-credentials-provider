<<<<<<< HEAD
# Jfrog Credential Provider
=======
# JFrog Credential Provider Sources
This directory has the sources for the JFrog Credential Provider.

## Building the JFrog Credential Provider
To build the JFrog Credential Provider, run the following command
```shell
./build-binary.sh
```

The resulting binaries will be in the `bin/` directory.


## PLUGIN REQUIREMENTS
To run the plugin 2 ENVs must exist:
`artifactory_url` pointing to the customer saas jfrog platform host (host name only excluding  https:// or training / signs)
example: my-jfrog-platform.jfrog.io
`aws_role_name` the name of the aws iam arn role that is set for the ec2 instances on which eks kubelet is running 
To provide the envs to kubelet, update them inside jfrog_provider.json which is being used by the project terraform

To check an ec2 role name, inside aws portal:
```
In the AWS Management Console, go to the EC2 Dashboard.
Select the instance you want to check
Search for IAM Role.```

THe IAM Role requires this permissions:
```
"Action": [
              "sts:GetCallerIdentity"
            ],
```
Optionally with this resource restriction:
```
"Resource": [
                "arn:aws:iam::<your account>:role/the iam role name"
            ]
```

## USED TOOLS / APIs
metadata local services:
get EC2 aws token

`curl -X PUT "http://169.254.169.254/latest/api/token" -H "X-aws-ec2-metadata-token-ttl-seconds: 21600"`

get temporary credentials for the EC2 role (uses the EC2 token)

`curl -o test -H "X-aws-ec2-metadata-token: $TOKEN" http://169.254.169.254/latest/meta-data/iam/security-credentials/<role name>```
temporary credentials response is:
`
  {
    "Code" : "Success",
    "LastUpdated" : "2024-12-08T15:38:52Z",
    "Type" : "AWS-HMAC",
    "AccessKeyId" : "xxx",
    "SecretAccessKey" : "xxx",
    "Token" : "xxx",
    "Expiration" : "2024-12-08T22:13:22Z"
  }
```

## LOGGING
plugin logs are written into: 
```tail -f /var/log/jfrog-credential-provider.log```

## JFROG SETTINGS 
For aws <> JFrog token exchange to work, the ec2 iam role needs to be mapped to JFrog artifactory user  
Tagging a jfrog user user with aws role:
`curl -XPUT -H "Content-type: application/json"  -H "Authorization: Bearer <TOKEN>"  https://<JFrog saas platform name>.jfrog.io/access/api/v1/aws/iam_role -d '{"username":"<jfrog user>", "iam_role": "<role arn>"}' -vvv`

verify JFrog user is tagged:
`curl -H "Content-type: application/json"  -H "Authorization: Bearer <TOKEN>"  https://<JFrog saas platform name>.jfrog.io/access/api/v1/aws/iam_role/<jfrog user> -vvv`

>>>>>>> c7906b7 (Initial Commit)
