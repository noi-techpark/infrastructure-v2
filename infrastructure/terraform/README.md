# TODO: add execution order
# TODO: outputs and inputs compute -> kubernetes
# TODO: verify that settings in Terraform are correct (i.e. share outputs from "compute" with all workspaces)
# TODO: addons unstable, maybe sleep before creating?

https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/using-eni.html#AvailableIpPerENI

Events:
  Type     Reason            Age   From               Message
  ----     ------            ----  ----               -------
  Warning  FailedScheduling  99s   default-scheduler  0/1 nodes are available: 1 Too many pods. preemption: 0/1 nodes are available: 1 No preemption victims found for incoming pod.


# TODO: hostname rabbitmq

https://github.com/hashicorp/terraform-provider-aws/issues/20404#issuecomment-895376217
