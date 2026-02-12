#!/bin/bash
# Suggest branch protection rules for Igor repository
# Prints recommended configuration WITHOUT applying it

echo "📋 Recommended Branch Protection for Igor"
echo ""
echo "These settings ensure code quality and review discipline."
echo "Run commands manually to apply (not automated to preserve workflow flexibility)."
echo ""

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Recommended command:"
echo ""
echo "gh repo edit \\"
echo "  --enable-squash-merge=true \\"
echo "  --enable-merge-commit=false \\"
echo "  --enable-rebase-merge=true \\"
echo "  --delete-branch-on-merge=true"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

echo "For branch protection rules (requires GitHub web UI or API):"
echo ""
echo "Settings → Branches → Add rule for 'main'"
echo ""
echo "Recommended protections:"
echo "  ✓ Require pull request reviews (1 approver)"
echo "  ✓ Require status checks to pass"
echo "    - Quality Checks (from CI)"
echo "  ✓ Require branches to be up to date"
echo "  ✓ Do not allow bypassing the above settings"
echo "  ✗ Do NOT require signed commits (optional)"
echo ""

echo "Note: These are recommendations, not requirements."
echo "Igor development can proceed without branch protection."
echo ""
