# GitHub Actions Workflows

This directory contains all CI/CD workflows for the Build Process Watcher project.

## 🧪 Test Workflows

### CI Tests (`ci-tests.yml`)
**Triggers:** Pull requests, pushes to main/develop branches

Main orchestrator workflow that runs all test suites together and provides a comprehensive test summary. This is the primary CI workflow that should pass before merging PRs.

**Components tested:**
- ✅ Backend Go tests
- ✅ Action TypeScript build
- ✅ Frontend validation
- ✅ End-to-end action tests

---

### Backend Tests (`test-backend.yml`)
**Triggers:** PRs, pushes (when backend files change)

Runs comprehensive Go backend testing:
- Unit tests with race detection
- Integration tests
- Test coverage reporting
- Performance benchmarks
- Go module verification

**Artifacts:**
- Coverage report (HTML)

---

### Action Build Tests (`test-action.yml`)
**Triggers:** PRs, pushes (when action files change)

Tests the GitHub Action TypeScript code:
- Builds TypeScript with `ncc`
- Verifies output files exist (`dist/index_with_backend.js`, `dist/cleanup.js`)
- TypeScript type checking
- Validates `action.yaml` metadata
- Checks script permissions

**Artifacts:**
- Built action files

---

### Frontend Validation (`test-frontend.yml`)
**Triggers:** PRs, pushes (when frontend files change)

Validates the static frontend:
- HTML syntax validation
- Firebase configuration validation
- Config injection testing
- JavaScript syntax checking
- Static asset verification
- URL rewrite structure validation

---

### E2E Action Tests (`test-e2e-action.yml`)
**Triggers:** PRs, pushes (when action files change)

End-to-end testing of the GitHub Action:
- **Local mode test**: Tests action without backend
- **Remote mode test**: Tests action with backend URL (dry run)
- **Script test**: Validates monitoring script syntax and execution
- **Composite actions**: Validates all action.yml files

**Artifacts:**
- Log files from test runs
- Backend debug logs

---

## 🚀 Deployment Workflows

### Deploy Backend (`deploy-backend.yml`)
**Triggers:** Pushes to main (when backend files change), manual dispatch

Deploys the Go backend to Google Cloud Run:
1. Runs all backend tests
2. Builds Docker container
3. Deploys to Cloud Run
4. Outputs service URL

**Required secrets:**
- `GOOGLE_CLOUD_PROJECT`
- `GOOGLE_APPLICATION_CREDENTIALS_JSON`
- `JWT_SECRET_KEY`
- `ADMIN_SECRET`

---

### Deploy Frontend (`deploy-frontend.yml`)
**Triggers:** Pushes to main (when frontend files change), manual dispatch

Deploys the static frontend to Firebase Hosting:
1. Injects configuration (`config.js`)
2. Deploys to Firebase Hosting

**Required secrets:**
- `BACKEND_URL`
- `FRONTEND_URL`
- `FIREBASE_SERVICE_ACCOUNT`
- `FIREBASE_PROJECT_ID`

---

## 📋 Required Secrets

### Google Cloud (Backend)
- `GOOGLE_CLOUD_PROJECT` - GCP project ID
- `GOOGLE_APPLICATION_CREDENTIALS_JSON` - Service account credentials
- `JWT_SECRET_KEY` - JWT signing key
- `ADMIN_SECRET` - Admin API secret

### Firebase (Frontend)
- `FIREBASE_SERVICE_ACCOUNT` - Firebase service account JSON
- `FIREBASE_PROJECT_ID` - Firebase project ID
- `BACKEND_URL` - Backend API URL
- `FRONTEND_URL` - Frontend hosting URL

---

## 🔄 Workflow Dependencies

```
ci-tests.yml (main)
├── test-backend.yml
├── test-action.yml
├── test-frontend.yml
└── test-e2e-action.yml
```

---

## 🎯 Development Workflow

1. **Create a PR** → All test workflows run automatically
2. **Review test results** → Check CI Tests workflow summary
3. **Fix any failures** → Push new commits to re-run tests
4. **Merge to main** → Deployment workflows run automatically

---

## 🛠️ Running Tests Locally

### Backend Tests
```bash
cd backend
make test              # Unit tests
make test-coverage     # With coverage report
make test-integration  # Integration tests
make benchmark        # Performance benchmarks
```

### Action Build
```bash
npm ci
npm run build
# Verify dist/index_with_backend.js and dist/cleanup.js exist
```

### Frontend Validation
```bash
cd frontend
# Install html-validate: npm install -g html-validate
html-validate public/*.html
# Validate firebase.json
cat firebase.json | jq .
```

### E2E Tests
```bash
# Build action
npm ci && npm run build

# Test in local mode (add to a test workflow)
# See test-e2e-action.yml for examples
```

---

## 📦 Artifacts

Test runs generate artifacts that are stored for 7-30 days:
- **Coverage reports** (Backend tests)
- **Build outputs** (Action build)
- **Test logs** (E2E tests)

Access artifacts from the GitHub Actions UI → Workflow run → Artifacts section.

---

## ⚙️ Configuration

### Customizing Test Paths
Edit the `paths` filter in each workflow to control when tests run:

```yaml
on:
  pull_request:
    paths:
      - 'backend/**'  # Only run when backend files change
```

### Adjusting Test Timeouts
Modify the `timeout-minutes` in job definitions (default: 360 minutes).

### Branch Protection
Recommended branch protection rules for `main`:
- ✅ Require status checks: `CI Tests / Test Summary`
- ✅ Require branches to be up to date
- ✅ Require linear history

---

## 🐛 Troubleshooting

### Tests fail on PR but pass locally
- Check GitHub Actions runner OS (ubuntu-latest)
- Verify all secrets are configured
- Check for race conditions
- Review artifact logs

### E2E tests fail
- Verify `dist/` files are committed
- Check `monitor_with_backend.sh` has correct permissions
- Review debug logs in artifacts

### Deployment fails
- Verify all required secrets are set
- Check GCP/Firebase permissions
- Review service account roles

---

## 📚 Additional Resources

- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [Google Cloud Run](https://cloud.google.com/run/docs)
- [Firebase Hosting](https://firebase.google.com/docs/hosting)









