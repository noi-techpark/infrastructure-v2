<!--
SPDX-FileCopyrightText: NOI Techpark <digital@noi.bz.it>

SPDX-License-Identifier: CC0-1.0
-->

# Velero

Velero is an open-source backup and restore solution for Kubernetes clusters. It simplifies the process of protecting your applications and persistent volumes by capturing and managing cluster-wide snapshots. Velero enables you to perform regular backups and efficiently restore your applications to a previous state.

## How Velero Works

1. **Snapshot Creation:** Velero creates consistent snapshots of your entire Kubernetes cluster, including all namespaces, resources, and persistent volumes.

2. **Backup Storage:** The snapshots are then securely stored in a designated backup storage location, which can be on-premises or in the cloud.

3. **Scheduled Backups:** Velero supports scheduled backups, allowing you to automate the process and ensure regular snapshots of your cluster. This is particularly useful for preventing data loss and ensuring disaster recovery preparedness.

4. **Restore Functionality:** In the event of a failure or data loss, Velero facilitates easy restoration by allowing you to recover your entire cluster or specific resources from a previously created backup.

Velero's simplicity and flexibility make it a valuable tool for Kubernetes administrators and DevOps teams looking to enhance the resilience of their applications and data in a Kubernetes environment.

## Scheduled Backups Setup

### Daily Backup

To set up scheduled daily backups with a retention policy of 30 days, use the following commands:

```bash
velero schedule create schedule-daily \
    --schedule="@every 24h" \
    --ttl 720h \
    --namespace velero-system \
    -o yaml | kubectl apply -f -
```

### Weekly Backup

To set up scheduled weekly backups with a retention policy of 90 days, use the following commands:

```bash
velero schedule create schedule-weekly \
    --schedule="@every 168h" \
    --ttl 2160h \
    --namespace velero-system \
    -o yaml | kubectl apply -f -
```

## List Resources

To list various common resources, you can use the following commands:

```bash
velero -n velero-system schedule get
velero -n velero-system backup get
velero -n velero-system backup-location get
```

## Manual Backup Example

To run a manual backup, use the following example:

```bash
velero -n velero-system backup create initial-backup
velero -n velero-system backup describe initial-backup
velero -n velero-system backup get initial-backup
```

Feel free to refer to the documentation links for a more comprehensive understanding of Velero and its features.

## Documentation Links

Explore the documentation for more details:

- [Velero Documentation](https://velero.io/docs/main/)
- [Backup Reference](https://velero.io/docs/main/backup-reference/)
- [Restore Reference](https://velero.io/docs/main/restore-reference/)
