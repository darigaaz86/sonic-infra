# Sonic Infrastructure

This repository sets up the Sonic infrastructure using **Terraform** for provisioning and **Ansible** for configuration.

---

## 📦 Prerequisites

Make sure the following tools are installed:

- [Terraform](https://developer.hashicorp.com/terraform/downloads)
- [Python 3](https://www.python.org/downloads/)
- [Ansible](https://docs.ansible.com/ansible/latest/installation_guide/intro_installation.html)
- SSH key with access to your EC2 instances (e.g. `~/.ssh/sonicKey.pem`)

---

## 🚀 Usage

### 1. Provision Infrastructure

```bash
cd terraform
terraform init
terraform apply
```

This creates the necessary cloud infrastructure (e.g., EC2 instances).

---

### 2. Generate Ansible Inventory

```bash
cd ../ansible
python3 generate_inventory.py
```

This generates `inventory.ini` using Terraform output.

---

### 3. Run Ansible Playbook

```bash
ansible-playbook -i inventory.ini playbook.yml \
  --private-key ~/.ssh/sonicKey.pem \
  -u ec2-user \
  --ssh-extra-args "-o StrictHostKeyChecking=no"
```

This sets up the software and services on the provisioned EC2 instances.

---

## 🐞 Debugging

### 🔍 View logs

SSH into your EC2 instance and run:

```bash
tail -f /var/log/sonic.log
```

### ⚙️ Check systemd service

To inspect how the Sonic service is running:

```bash
systemctl cat sonicd
```

---

## 🧼 Cleanup

To destroy all provisioned infrastructure:

```bash
cd terraform
terraform destroy
```

---

## 📁 Project Structure

```
.
├── terraform/              # Terraform configs for provisioning infrastructure
└── ansible/                # Ansible playbook and inventory generation script
```

---

## 📬 Support

Open an issue in this repository if you have any questions or need help.
