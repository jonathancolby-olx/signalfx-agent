# Fargate Deployment

## Create Task Definition
To deploy the agent on AWS Elastic Container Service (ECS) you must first
create the agent task definition.  To do this using the web admin console:

 1. Prepare task definition. You can either take signalfx-agent container
 	definition from the file [signalfx-agent-fargate-task.json](./signalfx-agent-fargate-task.json) or add
	your existing fargate container definitions to the file.
	- Make sure all your fargate containers to monitor has docker labels to specify ports to be monitored.
```json
	"dockerLabels": {
		"agent.signalfx.com.port.6379": "true",
		"agent.signalfx.com.config.6379.intervalSeconds": "1"
	}
```
 2. Go to your ECS web admin console and go to the "Task Definitions" tab.
 3. Click on "Create new Task Definition".
 4. Select the big "FARGATE" square and click "Next step".
 5. Scroll to the bottom of the page and click on "Configure via JSON".
 6. Paste in the contents of the file you prepared above and click "Save".
 7. Click on the "signalfx-agent" container definition under "Container
	Definitions" and find the section on environment variables.
 8. Change the value of the envvar `ACCESS_TOKEN` to the access token of the
	SignalFx organization to which you wish to send metrics.
 8. Click "Update" and finally "Create" at the bottom of the task definition
	input form to create the task definition.

You can also do this with the AWS CLI tool by issuing the following command:

`aws ecs register-task-definition --cli-input-json file:///path/to/signalfx-agent-fargate-task.json`

## Launching the Agent
The agent is designed to be run as a sidecar in a task with fargate containers
to be monitored.

To create an agent service from the ECS web admin console:

 1. Go to your cluster in the web admin
 2. Click on the "Services" tab.
 3. Click "Create" at the top of the tab.
 4. Select:
     - `Launch Type` -> `FARGATE`
	 - `Task Definition (Family)` -> `signalfx-fargate`
	 - `Task Definition (Revision)` -> `1` (or whatever the latest is in your case)
	 - `Service Name` -> `signalfx-fargate` (or any good name that explains you service)
     - `Task Definition (Revision)` -> `1` (or whatever the latest is in your case)
     - `Number of tasks` is also required for fargate service configuration
 5. Leave everything else at default and click "Next step"
 6. The second step is configuring network. Fargate requires to run with `awsvpc`
    network type.
    After providing all required network settings, click "Next step"
 7. Leave everything on this next page at their defaults and click "Next step".
 8. Click "Create Service" and the agent should be deployed with other fargate
    containers. As all the containers and the agent startup, you should see
    infrastructure and docker metrics flowing soon.


## Configuration

The main technique for configuring the agent is to have a config file
downloaded from the network using curl in the agent container's initialization
script.  By default it pulls from [the config file in our Github
repository](./agent.yaml) that provides a basic config that might suffice for
basic monitoring cases.  If you wish to provide a more complex config file you
can set the `CONFIG_URL` env var in the agent task definition to the URL of the
config file.  This location must be accessible from the ECS cluster.

The default config supports various environment variable overrides, which you
can set in the environment variable section of the agent task definition.  See
[agent.yaml](./agent.yaml) for details (hint: it is the config values that are
of the form `{"#from": "env:VARNAME"...}`).
