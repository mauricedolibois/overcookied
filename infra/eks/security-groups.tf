# Security Group for EKS Cluster
resource "aws_security_group" "cluster" {
  name_prefix = "${local.cluster_name}-cluster-"
  description = "Security group for EKS cluster control plane"
  vpc_id      = local.vpc_id

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "${local.cluster_name}-cluster-sg"
  }
}

# Security Group for EKS Nodes
resource "aws_security_group" "nodes" {
  name_prefix = "${local.cluster_name}-nodes-"
  description = "Security group for EKS worker nodes"
  vpc_id      = local.vpc_id

  # Allow nodes to communicate with each other
  ingress {
    from_port = 0
    to_port   = 65535
    protocol  = "tcp"
    self      = true
  }

  # Allow pods to communicate with each other
  ingress {
    from_port = 0
    to_port   = 65535
    protocol  = "udp"
    self      = true
  }

  # Allow EKS control plane to communicate with nodes
  ingress {
    from_port       = 443
    to_port         = 443
    protocol        = "tcp"
    security_groups = [aws_security_group.cluster.id]
  }

  # Allow kubelet API from control plane
  ingress {
    from_port       = 10250
    to_port         = 10250
    protocol        = "tcp"
    security_groups = [aws_security_group.cluster.id]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name                                        = "${local.cluster_name}-nodes-sg"
    "kubernetes.io/cluster/${local.cluster_name}" = "owned"
  }
}

# Security Group Rule: Allow nodes to reach control plane
resource "aws_security_group_rule" "cluster_ingress_nodes" {
  type                     = "ingress"
  from_port                = 443
  to_port                  = 443
  protocol                 = "tcp"
  source_security_group_id = aws_security_group.nodes.id
  security_group_id        = aws_security_group.cluster.id
  description              = "Allow nodes to communicate with control plane"
}
