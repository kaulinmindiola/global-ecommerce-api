<!-- .github/pull_request_template.md -->

## 📋 Description

<!-- Describe your changes clearly and concisely -->

**What:** 
**Why:** 
**How:** 

Closes # <!-- Issue number -->

---

## 🔀 Type of Change

- [ ] 🐛 Bug fix (non-breaking change that fixes an issue)
- [ ] ✨ New feature (non-breaking change that adds functionality)
- [ ] 💥 Breaking change (fix or feature that would cause existing functionality to change)
- [ ] 🏗️ Refactoring (no functional changes, just code improvements)
- [ ] 📝 Documentation update
- [ ] 🔧 Configuration change
- [ ] 🧪 Test improvements

---

## ✅ Checklist

### Code Quality
- [ ] Code follows the project's style guidelines (`make lint` passes)
- [ ] Code is self-documenting with clear naming
- [ ] No magic numbers or hardcoded values
- [ ] Error handling is comprehensive
- [ ] No TODO comments left (or they're tracked as issues)

### Testing
- [ ] Unit tests added/updated for new functionality
- [ ] Integration tests added/updated if needed
- [ ] All existing tests pass (`make test` succeeds)
- [ ] Code coverage maintained above 80%

### Documentation
- [ ] Swagger annotations updated for new/modified endpoints
- [ ] README updated if needed
- [ ] API contract changes are backward compatible (or documented as breaking)

### Security
- [ ] No sensitive data (passwords, API keys) in code
- [ ] Input validation for all new endpoints
- [ ] Authentication/authorization properly applied

---

## 🧪 Testing Instructions

<!-- How to test this PR manually -->

```bash
# 1. Start the development environment
make dev

# 2. Run specific test
go test -v ./...

# 3. Test the endpoint manually
curl -X GET http://localhost:8080/api/v1/...
```

---

## 📸 Screenshots (if applicable)

<!-- Add screenshots for UI changes or Swagger UI updates -->

---

## 🔗 Related Issues/PRs

- Related to #
- Depends on #
- Blocks #

---

## ⚠️ Notes for Reviewers

<!-- Anything specific reviewers should focus on -->
