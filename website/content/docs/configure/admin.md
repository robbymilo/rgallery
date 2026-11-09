---
title: Admin
---

# Admin

The admin UI allows administrators to initiate scans, create users, create API keys, and review scan errors.

## Scan errors

Open **Admin → Scan errors** to see files that failed during scanning. The table is paginated and provides a reset action for each filepath. **Reset all** clears every recorded error so those files are eligible for the next scan.

Viewers cannot access this page or its API endpoints.

## API keys

> API keys are granted admin access.

To create an API key:

1. Click the admin link in the nav.
1. Click **Add API key**.
1. Enter a name for the API key.
1. Click **Add key**.

Save the API key to your secrets manager as it will never be shown again.

## How to schedule scans

To initiate a scan daily at, for example, 00:30:

1. Create an API key
1. Create a cron job on a server, such as:

```shell
30 0 * * * curl -H 'api-key: $(API_KEY)' 'https://<replace-with-rgallery-url>/api/scan'
```

> You may need to restart the cron service - refer to your distribution's documentation.
