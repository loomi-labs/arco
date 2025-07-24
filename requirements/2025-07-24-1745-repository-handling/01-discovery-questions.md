# Discovery Questions - Repository Handling

## Q1: Will users interact with repository management through the desktop application UI?
**Default if unknown:** Yes (repository management typically requires visual interface for configuration and monitoring)

## Q2: Should repository operations work when the user is offline?
**Default if unknown:** No (BorgBase cloud operations require internet connectivity for API calls)

## Q3: Will repository management handle sensitive data that needs encryption?
**Default if unknown:** Yes (SSH keys, passwords, and repository access credentials are highly sensitive)

## Q4: Do users currently manage repositories through manual BorgBase account setup?
**Default if unknown:** Yes (without cloud integration, users likely manually configure BorgBase repositories)

## Q5: Should repository operations integrate with existing backup scheduling and monitoring?
**Default if unknown:** Yes (repositories are storage destinations for backup operations, integration is essential)