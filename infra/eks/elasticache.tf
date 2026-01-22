# ElastiCache Valkey for Game Queue/Matchmaking
# This provides shared state across all backend pods

# Subnet Group for ElastiCache
resource "aws_elasticache_subnet_group" "valkey" {
  name       = "${var.project_name}-valkey-subnet"
  subnet_ids = local.public_subnet_ids

  tags = {
    Name = "${var.project_name}-valkey-subnet"
  }
}

# Security Group for ElastiCache
resource "aws_security_group" "valkey" {
  name        = "${var.project_name}-valkey-sg"
  description = "Security group for Valkey cache"
  vpc_id      = local.vpc_id

  # Allow traffic from EKS cluster security group (used by nodes)
  ingress {
    description     = "Valkey from EKS cluster nodes"
    from_port       = 6379
    to_port         = 6379
    protocol        = "tcp"
    security_groups = [aws_eks_cluster.main.vpc_config[0].cluster_security_group_id]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "${var.project_name}-valkey-sg"
  }
}

# ElastiCache Valkey Replication Group (required for Valkey engine)
resource "aws_elasticache_replication_group" "valkey" {
  replication_group_id = "${var.project_name}-valkey"
  description          = "Valkey cache for Overcookied matchmaking"
  
  engine               = "valkey"
  engine_version       = "8.0"
  node_type            = var.valkey_node_type
  port                 = 6379
  parameter_group_name = "default.valkey8"
  
  # Single node setup (no replicas for cost savings)
  num_cache_clusters   = 1
  
  subnet_group_name    = aws_elasticache_subnet_group.valkey.name
  security_group_ids   = [aws_security_group.valkey.id]

  # Disable automatic failover for single node
  automatic_failover_enabled = false
  
  # Enable automatic minor version upgrades
  auto_minor_version_upgrade = true

  tags = {
    Name = "${var.project_name}-valkey"
  }
}
