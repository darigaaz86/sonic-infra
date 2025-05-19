provider "aws" {
  profile = "AwsSandboxAdmin-945430081584"
  region  = "ap-southeast-1"
}

variable "instance_count" { default = 8 }

resource "aws_instance" "sonic_node" {
  count           = var.instance_count
  ami             = "ami-0afc7fe9be84307e4"
  instance_type   = "i3en.xlarge"
  key_name        = "sonicKey"
  security_groups = ["default"]

  tags = {
    Name = count.index < 5 ? "validator-${count.index + 1}" : "rpc-${count.index - 4}"
    Role = count.index < 5 ? "validator" : "rpc"
  }

  provisioner "local-exec" {
    command = "echo ${self.public_ip} >> hosts.txt"
  }
}

# Data source to fetch default VPC
data "aws_vpc" "default" {
  default = true
}

# Data source to fetch subnets in default VPC (public)
data "aws_subnets" "public" {
  filter {
    name   = "vpc-id"
    values = [data.aws_vpc.default.id]
  }
}

resource "aws_lb" "rpc_nlb" {
  name               = "rpc-nlb"
  internal           = false
  load_balancer_type = "network"
  subnets            = data.aws_subnets.public.ids
}

resource "aws_lb_target_group" "rpc_target_group" {
  name     = "rpc-target-group"
  port     = 8545
  protocol = "TCP"
  vpc_id   = data.aws_vpc.default.id

  health_check {
    protocol            = "TCP"
    port                = "8545"
    healthy_threshold   = 2
    unhealthy_threshold = 2
    timeout             = 3
    interval            = 10
  }
}

# Attach only RPC instances (count 5 to 7)
resource "aws_lb_target_group_attachment" "rpc_targets" {
  count            = 3
  target_group_arn = aws_lb_target_group.rpc_target_group.arn
  target_id        = aws_instance.sonic_node[count.index + 5].id
  port             = 8545
}

resource "aws_lb_listener" "rpc_listener" {
  load_balancer_arn = aws_lb.rpc_nlb.arn
  port              = 8545
  protocol          = "TCP"

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.rpc_target_group.arn
  }
}

output "node_ips" {
  value = aws_instance.sonic_node[*].public_ip
}

output "rpc_nlb_dns" {
  value = aws_lb.rpc_nlb.dns_name
}
