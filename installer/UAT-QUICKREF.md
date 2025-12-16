# UAT Quick Reference Card

**Print this page for quick reference during UAT sessions**

---

## Time Targets

| Task | Target | Critical? |
|------|--------|-----------|
| Windows Installation | < 10 minutes | YES |
| Linux Installation | < 15 minutes | YES |
| Error Recovery | < 5 minutes | YES |
| Service Start | < 5 minutes | NO |
| Uninstallation | < 5 minutes | NO |

---

## Pre-Session Checklist

- [ ] Downloaded installer package
- [ ] Test system is clean (no previous installation)
- [ ] Have timer ready
- [ ] Have UAT-RESULTS.md open for notes
- [ ] Know your technical level (Non-technical/Basic/Intermediate/Advanced)

---

## During Testing: What to Look For

### Installation Phase
✅ **Good Signs**:
- Each step is obvious
- Default options make sense
- Progress bar advances smoothly
- Success message is clear
- You know where files installed

❌ **Bad Signs**:
- Unclear what to click next
- Error messages with no explanation
- Installation hangs or freezes
- Don't know if it succeeded

### Configuration Phase
✅ **Good Signs**:
- Easy to find .env file
- Comments explain each setting
- Example values help
- Can save changes easily

❌ **Bad Signs**:
- Can't find configuration file
- No explanation of settings
- Don't know what values to enter
- Permission errors when saving

### Service Start Phase
✅ **Good Signs**:
- Clear how to start service
- Status shows "running" or "active"
- Can find logs easily
- Logs show success messages

❌ **Bad Signs**:
- Don't know how to start service
- Service fails with no error
- Can't tell if running
- Can't find logs

### Error Recovery Phase
✅ **Good Signs**:
- Error message in plain English
- Error tells you WHAT went wrong
- Error tells you HOW to fix it
- Can fix and retry quickly

❌ **Bad Signs**:
- Cryptic error codes
- No explanation of problem
- No guidance on fix
- Have to Google the error

---

## Windows-Specific Checks

### Installation
- [ ] UAC prompt is expected (needs admin)
- [ ] Installer shows in Add/Remove Programs
- [ ] Desktop shortcut works (if selected)
- [ ] Start Menu folder created
- [ ] Service appears in services.msc

### Files to Verify
```
C:\Program Files\CanvusLocalLLM\
├── CanvusLocalLLM.exe
├── .env.example
├── LICENSE.txt
├── README.txt
└── Uninstall.exe
```

### Quick Commands (PowerShell)
```powershell
# Check if installed
Test-Path "$env:ProgramFiles\CanvusLocalLLM\CanvusLocalLLM.exe"

# Check service
Get-Service CanvusLocalLLM

# Start service
Start-Service CanvusLocalLLM

# Check service status
Get-Service CanvusLocalLLM | Select Status
```

---

## Linux-Specific Checks

### Installation (Debian)
- [ ] Package installs without dependency errors
- [ ] Service file created in /etc/systemd/system/
- [ ] User `canvusllm` created
- [ ] Files owned by correct user
- [ ] Service enabled for auto-start

### Files to Verify
```
/opt/canvuslocallm/
├── canvuslocallm (binary)
├── .env.example
├── .env (created during setup)
├── LICENSE.txt
└── README.txt
```

### Quick Commands (Bash)
```bash
# Check if installed (deb)
dpkg -l | grep canvuslocallm

# Check if installed (tarball)
ls -la /opt/canvuslocallm/

# Check service status
systemctl status canvuslocallm

# Start service
sudo systemctl start canvuslocallm

# View logs
journalctl -u canvuslocallm -n 50

# Check user created
getent passwd canvusllm
```

---

## Error Scenarios to Test

### Scenario A: Missing .env
1. Delete or rename .env file
2. Try to start service
3. **Expect**: Clear error about missing config
4. **Expect**: Instructions on how to create .env
5. **Expect**: Can recover in < 5 minutes

### Scenario B: Invalid API Key
1. Put garbage in OPENAI_API_KEY
2. Start service
3. Make API request (or wait for first auto-request)
4. **Expect**: Log shows authentication error
5. **Expect**: Error identifies which API key is wrong
6. **Expect**: Can fix and restart in < 5 minutes

### Scenario C: Port Conflict
1. Start another service on port 1234
2. Try to start CanvusLocalLLM
3. **Expect**: Error about port in use
4. **Expect**: Suggestion to check .env PORT setting
5. **Expect**: Can change port and restart in < 5 minutes

---

## Rating Scale for Errors

**5 - Excellent**: Error tells me exactly what's wrong and how to fix it
**4 - Good**: Error is clear, fix is obvious or easily found
**3 - Fair**: Error is understandable but I had to think about fix
**2 - Poor**: Error is confusing, had to try multiple things
**1 - Terrible**: Cryptic error, no idea what's wrong or how to fix

---

## Post-Session Questions

Ask yourself after each scenario:

1. **Clarity**: Did I know what to do at each step?
   - [ ] Always  [ ] Usually  [ ] Sometimes  [ ] Rarely

2. **Confidence**: Did I feel in control of the process?
   - [ ] Always  [ ] Usually  [ ] Sometimes  [ ] Rarely

3. **Frustration**: How frustrated was I? (1=not at all, 5=very)
   - [ ] 1  [ ] 2  [ ] 3  [ ] 4  [ ] 5

4. **Documentation**: Was README/help sufficient?
   - [ ] Yes, didn't need external help
   - [ ] Mostly, only minor questions
   - [ ] No, had to Google things
   - [ ] No, needed expert help

5. **Recommendation**: Would I recommend this to a colleague?
   - [ ] Yes, definitely
   - [ ] Yes, with minor reservations
   - [ ] Maybe, has some issues
   - [ ] No, too frustrating

---

## Quick Issue Logging Format

When you find a problem, note:

**What**: [One sentence describing the issue]
**Where**: [Which scenario/step]
**Expected**: [What you thought should happen]
**Actual**: [What actually happened]
**Impact**: [How bad is it? Critical/Moderate/Minor]
**Workaround**: [Could you get past it? How?]

Example:
```
What: Error message says "Error 500" with no details
Where: Scenario 6 - Invalid API Key
Expected: Should say which API key is wrong
Actual: Generic error code
Impact: Moderate - had to check all API keys
Workaround: Checked logs to find real error
```

---

## Critical vs Non-Critical Issues

### Critical = Release Blocker
- Can't complete installation
- Service won't start and error doesn't help
- Data loss or corruption
- Security vulnerability
- Can't recover from error in reasonable time
- Misleading error that causes wrong action

### Non-Critical = Should Fix
- Confusing but not blocking
- Extra steps needed but works
- Typos in documentation
- UI/UX annoyances
- Missing nice-to-have features

### Enhancement = Could Add Later
- Feature requests
- Convenience improvements
- Better default values
- Additional documentation
- Cosmetic improvements

---

## Session Completion Checklist

Before ending your UAT session:

- [ ] Recorded all times for scenarios tested
- [ ] Documented all issues found
- [ ] Rated error message quality
- [ ] Filled out satisfaction scores
- [ ] Answered recommendation question
- [ ] Wrote down improvement suggestions
- [ ] Noted what worked well
- [ ] Noted what was confusing

---

## Emergency Contacts

**Issue tracker**: Create issue in `.beads/issues.jsonl` or GitHub
**Documentation**: See `installer/UAT-GUIDE.md` for detailed scenarios
**Results**: Record in `installer/UAT-RESULTS.md`

---

**Remember**: Your feedback as a non-technical user is EXTREMELY valuable. Don't hold back on criticism - we want to know what's confusing or frustrating!

**Thank you for testing!**
