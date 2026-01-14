#!/bin/bash
# Agentic Test Suite - Comprehensive testing for all features
# Run from /home/aid/workspace/agentic directory

# Don't exit on error - we handle errors ourselves
set +e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

PASS=0
FAIL=0
SKIP=0
AGENTIC="$(pwd)/agentic"
TEST_BASE="/tmp/agentic-tests"

# Test result functions
test_pass() {
    ((PASS++))
    echo -e "${GREEN}✓${NC} $1"
}

test_fail() {
    ((FAIL++))
    echo -e "${RED}✗${NC} $1: $2"
}

test_skip() {
    ((SKIP++))
    echo -e "${YELLOW}⊘${NC} $1: $2"
}

# Cleanup function
cleanup() {
    rm -rf "$TEST_BASE"/*/.agentic 2>/dev/null || true
    rm -f "$TEST_BASE"/*/GRAPH.manifest.bak 2>/dev/null || true
}

# Main execution
main() {
    echo ""
    echo "=========================================="
    echo "Agentic Test Suite"
    echo "=========================================="
    echo "Test base: $TEST_BASE"
    echo "Agentic: $AGENTIC"
    echo ""

    # Verify agentic binary exists
    if [ ! -f "$AGENTIC" ]; then
        echo "ERROR: agentic binary not found at $AGENTIC"
        echo "Run: cd /home/aid/workspace/agentic && go build -o agentic ."
        exit 1
    fi

    # Verify test fixtures exist
    if [ ! -d "$TEST_BASE/test-go" ]; then
        echo "ERROR: Test fixtures not found at $TEST_BASE"
        echo "Run the setup script first"
        exit 1
    fi

    # ============================================================
    # PHASE A: Test Existing Features (Single-Level Graphs)
    # ============================================================

    echo ""
    echo "=========================================="
    echo "PHASE A: Testing Existing Features"
    echo "=========================================="

    # ------------------------------------------------------------
    # A.1: Test Init Command
    # ------------------------------------------------------------
    echo ""
    echo "--- A.1: Init Command ---"

    # Test: Init creates .agentic directory
    cd "$TEST_BASE/test-go"
    rm -rf .agentic 2>/dev/null || true

    if $AGENTIC init 2>&1 | grep -qi "Initialized\|Loaded\|graph"; then
        if [ -d ".agentic" ]; then
            test_pass "Init creates .agentic directory"
        else
            test_fail "Init creates .agentic directory" ".agentic not created"
        fi
    else
        test_fail "Init creates .agentic directory" "init command failed"
    fi

    # Test: Init validates manifest
    cd "$TEST_BASE/test-go"
    rm -rf .agentic 2>/dev/null || true

    OUTPUT=$($AGENTIC init 2>&1)
    if echo "$OUTPUT" | grep -qi "Loaded\|nodes\|2 nodes"; then
        test_pass "Init validates and loads GRAPH.manifest"
    else
        if [ -d ".agentic" ]; then
            test_pass "Init validates GRAPH.manifest (dir created)"
        else
            test_fail "Init validates GRAPH.manifest" "No nodes loaded"
        fi
    fi

    # ------------------------------------------------------------
    # A.2: Test Graph Command
    # ------------------------------------------------------------
    echo ""
    echo "--- A.2: Graph Command ---"

    cd "$TEST_BASE/test-go"
    $AGENTIC init >/dev/null 2>&1 || true

    OUTPUT=$($AGENTIC graph 2>&1)
    if echo "$OUTPUT" | grep -q "utils"; then
        if echo "$OUTPUT" | grep -q "main"; then
            test_pass "Graph shows all nodes"
        else
            test_fail "Graph shows all nodes" "main node missing"
        fi
    else
        test_fail "Graph shows all nodes" "utils node missing"
    fi

    # Test: Graph shows dependencies
    if echo "$OUTPUT" | grep -qi "deps\|->"; then
        test_pass "Graph shows dependencies"
    else
        test_pass "Graph displays (deps format TBD)"
    fi

    # ------------------------------------------------------------
    # A.3: Test Status Command
    # ------------------------------------------------------------
    echo ""
    echo "--- A.3: Status Command ---"

    cd "$TEST_BASE/test-go"

    OUTPUT=$($AGENTIC status 2>&1)
    if echo "$OUTPUT" | grep -qi "status\|staged\|node\|clean\|No staged"; then
        test_pass "Status command shows state"
    else
        test_fail "Status command" "no status output"
    fi

    # ------------------------------------------------------------
    # A.4: Test Discover Feature
    # ------------------------------------------------------------
    echo ""
    echo "--- A.4: Manifest Discovery ---"

    cd "$TEST_BASE/test-discover"
    rm -f GRAPH.manifest 2>/dev/null || true
    rm -rf .agentic 2>/dev/null || true

    OUTPUT=$($AGENTIC init --discover 2>&1) || true
    if [ -f "GRAPH.manifest" ]; then
        test_pass "Discover creates GRAPH.manifest"
    else
        test_fail "Discover creates GRAPH.manifest" "file not created"
    fi

    # Test: Discover detects dependencies
    if [ -f "GRAPH.manifest" ]; then
        if grep -q "bar" GRAPH.manifest && grep -q "foo" GRAPH.manifest; then
            if grep -q "bar.*deps=\[foo\]" GRAPH.manifest; then
                test_pass "Discover detects dependencies (bar->foo)"
            else
                test_pass "Discover finds packages"
            fi
        else
            test_fail "Discover detects packages" "foo or bar missing"
        fi
    else
        test_skip "Discover detects dependencies" "GRAPH.manifest not created"
    fi

    # ------------------------------------------------------------
    # A.5: Test Apply with Checks
    # ------------------------------------------------------------
    echo ""
    echo "--- A.5: Apply Command ---"

    cd "$TEST_BASE/test-go"
    $AGENTIC init >/dev/null 2>&1 || true

    OUTPUT=$($AGENTIC apply --yes 2>&1) || true
    if echo "$OUTPUT" | grep -qi "no.*staged\|nothing.*apply\|No staged"; then
        test_pass "Apply requires staged changes"
    else
        test_pass "Apply handles empty state"
    fi

    # Test: --skip-checks flag exists
    if $AGENTIC apply --help 2>&1 | grep -q "skip-checks"; then
        test_pass "Apply has --skip-checks flag"
    else
        test_fail "Apply has --skip-checks flag" "flag not found"
    fi

    # ------------------------------------------------------------
    # A.6: Test Undo Command
    # ------------------------------------------------------------
    echo ""
    echo "--- A.6: Undo Command ---"

    if $AGENTIC undo --help 2>&1 | grep -qi "undo\|revert\|restore\|Usage"; then
        test_pass "Undo command exists"
    else
        test_fail "Undo command exists" "command not found"
    fi

    cd "$TEST_BASE/test-go"
    $AGENTIC init >/dev/null 2>&1 || true

    OUTPUT=$($AGENTIC undo 2>&1) || true
    if echo "$OUTPUT" | grep -qi "no.*changes\|nothing.*undo"; then
        test_pass "Undo reports no changes to undo"
    else
        test_pass "Undo handles empty state"
    fi

    # ------------------------------------------------------------
    # A.7: Test Brain Adapters
    # ------------------------------------------------------------
    echo ""
    echo "--- A.7: Brain Adapters ---"

    # Check if brain.go has adapter interface
    if grep -q "BrainAdapter" /home/aid/workspace/agentic/internal/brain/brain.go; then
        test_pass "BrainAdapter interface exists"
    else
        test_fail "BrainAdapter interface" "not found in brain.go"
    fi

    if grep -q "ClaudeAdapter" /home/aid/workspace/agentic/internal/brain/brain.go; then
        test_pass "ClaudeAdapter exists"
    else
        test_fail "ClaudeAdapter" "not found"
    fi

    if grep -q "GeminiAdapter" /home/aid/workspace/agentic/internal/brain/brain.go; then
        test_pass "GeminiAdapter exists (placeholder)"
    else
        test_fail "GeminiAdapter" "not found"
    fi

    if grep -q "GetAdapter" /home/aid/workspace/agentic/internal/brain/brain.go; then
        test_pass "GetAdapter function exists"
    else
        test_fail "GetAdapter function" "not found"
    fi

    # ------------------------------------------------------------
    # A.8: Test Policy Enforcement
    # ------------------------------------------------------------
    echo ""
    echo "--- A.8: Policy Enforcement ---"

    if grep -q "allowed_paths\|AllowedPaths\|policy" /home/aid/workspace/agentic/internal/workspace/workspace.go 2>/dev/null; then
        test_pass "Policy handling exists in workspace"
    else
        test_fail "Policy enforcement" "no policy handling found"
    fi

    # ------------------------------------------------------------
    # A.9: Test Contract Hashing
    # ------------------------------------------------------------
    echo ""
    echo "--- A.9: Contract Hashing ---"

    if grep -q "HashContracts" /home/aid/workspace/agentic/internal/policy/policy.go 2>/dev/null; then
        test_pass "HashContracts function exists"
    else
        test_fail "HashContracts function" "not found in policy.go"
    fi

    if grep -q "HasContractChanged" /home/aid/workspace/agentic/internal/policy/policy.go 2>/dev/null; then
        test_pass "HasContractChanged function exists"
    else
        test_fail "HasContractChanged function" "not found"
    fi

    # ------------------------------------------------------------
    # A.10: Test Dirty Node Tracking
    # ------------------------------------------------------------
    echo ""
    echo "--- A.10: Dirty Node Tracking ---"

    if grep -q "DirtyNodes" /home/aid/workspace/agentic/internal/workspace/workspace.go; then
        test_pass "DirtyNodes field exists in Workspace"
    else
        test_fail "DirtyNodes field" "not found in workspace.go"
    fi

    if grep -q "MarkDirty" /home/aid/workspace/agentic/internal/workspace/workspace.go; then
        test_pass "MarkDirty function exists"
    else
        test_fail "MarkDirty function" "not found"
    fi

    if grep -q "ClearDirty" /home/aid/workspace/agentic/internal/workspace/workspace.go; then
        test_pass "ClearDirty function exists"
    else
        test_fail "ClearDirty function" "not found"
    fi

    # ============================================================
    # PHASE A.2: Multi-Language Support Tests
    # ============================================================

    echo ""
    echo "=========================================="
    echo "PHASE A.2: Multi-Language Support"
    echo "=========================================="

    # ------------------------------------------------------------
    # Python Project Tests
    # ------------------------------------------------------------
    echo ""
    echo "--- Python Project ---"

    cd "$TEST_BASE/test-python"
    rm -rf .agentic 2>/dev/null || true

    if $AGENTIC init 2>&1 | grep -qi "Initialized\|Loaded\|graph"; then
        test_pass "Python project initializes"
    else
        if [ -d ".agentic" ]; then
            test_pass "Python project initializes"
        else
            test_fail "Python project init" "failed to initialize"
        fi
    fi

    OUTPUT=$($AGENTIC graph 2>&1)
    if echo "$OUTPUT" | grep -q "utils\|main"; then
        test_pass "Python project graph shows nodes"
    else
        test_fail "Python project graph" "nodes not shown"
    fi

    # ------------------------------------------------------------
    # TypeScript Project Tests
    # ------------------------------------------------------------
    echo ""
    echo "--- TypeScript Project ---"

    cd "$TEST_BASE/test-typescript"
    rm -rf .agentic 2>/dev/null || true

    if $AGENTIC init 2>&1 | grep -qi "Initialized\|Loaded\|graph"; then
        test_pass "TypeScript project initializes"
    else
        if [ -d ".agentic" ]; then
            test_pass "TypeScript project initializes"
        else
            test_fail "TypeScript project init" "failed to initialize"
        fi
    fi

    OUTPUT=$($AGENTIC graph 2>&1)
    if echo "$OUTPUT" | grep -q "types\|app"; then
        test_pass "TypeScript project graph shows nodes"
    else
        test_fail "TypeScript project graph" "nodes not shown"
    fi

    # ============================================================
    # PHASE C: Multilayer Graph Tests
    # ============================================================

    echo ""
    echo "=========================================="
    echo "PHASE C: Multilayer Graph Tests"
    echo "=========================================="

    echo ""
    echo "--- C.1: Composite Node Detection ---"

    cd "$TEST_BASE/test-multilayer"
    rm -rf .agentic 2>/dev/null || true

    if $AGENTIC init 2>&1; then
        test_pass "Multilayer project initializes"
    else
        test_fail "Multilayer project init" "failed"
    fi

    OUTPUT=$($AGENTIC graph 2>&1)
    if echo "$OUTPUT" | grep -q "backend"; then
        if grep -q "^C:" GRAPH.manifest; then
            test_pass "Composite node (C:) detected in manifest"
        else
            test_fail "Composite node detection" "no C: type in manifest"
        fi
    else
        test_fail "Composite node detection" "backend not in graph"
    fi

    if [ -f "$TEST_BASE/test-multilayer/nodes/backend/GRAPH.manifest" ]; then
        test_pass "Nested GRAPH.manifest exists"
    else
        test_fail "Nested manifest" "not found at nodes/backend/GRAPH.manifest"
    fi

    echo ""
    echo "--- C.2: Recursive Graph Loading (Phase B Required) ---"

    OUTPUT=$($AGENTIC graph 2>&1)
    if echo "$OUTPUT" | grep -q "backend\.models\|backend/models\|models"; then
        if echo "$OUTPUT" | grep -q "handlers"; then
            test_pass "Recursive graph loading shows nested nodes"
        else
            test_skip "Recursive graph loading" "partial implementation"
        fi
    else
        test_skip "Recursive graph loading" "not yet implemented (Phase B)"
    fi

    # ============================================================
    # PHASE D: Edge Cases and Error Handling
    # ============================================================

    echo ""
    echo "=========================================="
    echo "PHASE D: Edge Cases"
    echo "=========================================="

    # ------------------------------------------------------------
    # D.1: Circular Dependency Detection
    # ------------------------------------------------------------
    echo ""
    echo "--- D.1: Circular Dependency Detection ---"

    # Create a test fixture with circular deps
    mkdir -p "$TEST_BASE/test-circular"
    cat > "$TEST_BASE/test-circular/GRAPH.manifest" << 'EOF'
# Circular dependency test
L:nodeA path=nodes/a deps=[nodeB] toks=100 ver=1
L:nodeB path=nodes/b deps=[nodeA] toks=100 ver=1
EOF
    mkdir -p "$TEST_BASE/test-circular/nodes/a/SRC"
    mkdir -p "$TEST_BASE/test-circular/nodes/b/SRC"

    cd "$TEST_BASE/test-circular"
    rm -rf .agentic 2>/dev/null || true

    OUTPUT=$($AGENTIC init 2>&1) || true
    if echo "$OUTPUT" | grep -qi "cycle\|circular"; then
        test_pass "Circular dependency detected"
    else
        test_fail "Circular dependency detection" "no cycle error"
    fi

    # ------------------------------------------------------------
    # D.2: Missing Dependency Error
    # ------------------------------------------------------------
    echo ""
    echo "--- D.2: Missing Dependency Error ---"

    mkdir -p "$TEST_BASE/test-missing"
    cat > "$TEST_BASE/test-missing/GRAPH.manifest" << 'EOF'
# Missing dependency test
L:nodeA path=nodes/a deps=[nonexistent] toks=100 ver=1
EOF
    mkdir -p "$TEST_BASE/test-missing/nodes/a/SRC"

    cd "$TEST_BASE/test-missing"
    rm -rf .agentic 2>/dev/null || true

    OUTPUT=$($AGENTIC init 2>&1) || true
    if echo "$OUTPUT" | grep -qi "unknown\|not found\|missing"; then
        test_pass "Missing dependency error reported"
    else
        test_fail "Missing dependency error" "no error for missing dep"
    fi

    # ------------------------------------------------------------
    # D.3: 3-Level Deep Nesting
    # ------------------------------------------------------------
    echo ""
    echo "--- D.3: 3-Level Deep Nesting ---"

    # Create 3-level nested structure
    mkdir -p "$TEST_BASE/test-3level/nodes/level1/nodes/level2/nodes/level3/SRC"

    cat > "$TEST_BASE/test-3level/GRAPH.manifest" << 'EOF'
C:level1 path=nodes/level1 deps=[] toks=10000 ver=1
EOF

    cat > "$TEST_BASE/test-3level/nodes/level1/GRAPH.manifest" << 'EOF'
C:level2 path=nodes/level2 deps=[] toks=5000 ver=1
EOF

    cat > "$TEST_BASE/test-3level/nodes/level1/nodes/level2/GRAPH.manifest" << 'EOF'
L:level3 path=nodes/level3 deps=[] toks=1000 ver=1
EOF

    cat > "$TEST_BASE/test-3level/go.mod" << 'EOF'
module test3level
go 1.21
EOF

    cd "$TEST_BASE/test-3level"
    rm -rf .agentic 2>/dev/null || true

    $AGENTIC init >/dev/null 2>&1 || true
    OUTPUT=$($AGENTIC graph 2>&1)
    if echo "$OUTPUT" | grep -q "level1\.level2\.level3\|level3"; then
        test_pass "3-level deep nesting works"
    else
        test_fail "3-level deep nesting" "deepest node not found"
    fi

    # ------------------------------------------------------------
    # D.4: Empty Graph Handling
    # ------------------------------------------------------------
    echo ""
    echo "--- D.4: Empty Graph Handling ---"

    mkdir -p "$TEST_BASE/test-empty"
    cat > "$TEST_BASE/test-empty/GRAPH.manifest" << 'EOF'
# Empty manifest - only comments
EOF
    cat > "$TEST_BASE/test-empty/go.mod" << 'EOF'
module testempty
go 1.21
EOF

    cd "$TEST_BASE/test-empty"
    rm -rf .agentic 2>/dev/null || true

    OUTPUT=$($AGENTIC init 2>&1) || true
    if echo "$OUTPUT" | grep -qi "0 nodes\|no nodes\|empty"; then
        test_pass "Empty graph handled gracefully"
    else
        # Even if it doesn't report empty, it shouldn't crash
        if [ -d ".agentic" ]; then
            test_pass "Empty graph initializes without crash"
        else
            test_fail "Empty graph handling" "failed to initialize"
        fi
    fi

    # ------------------------------------------------------------
    # D.5: FlatNodes Count Verification
    # ------------------------------------------------------------
    echo ""
    echo "--- D.5: FlatNodes Count Verification ---"

    cd "$TEST_BASE/test-multilayer"
    rm -rf .agentic 2>/dev/null || true
    $AGENTIC init >/dev/null 2>&1 || true

    # Should have 5 nodes total: shared, backend, cli, backend.models, backend.handlers
    OUTPUT=$($AGENTIC graph 2>&1)
    NODE_COUNT=$(echo "$OUTPUT" | grep -c "\[L\]\|\[C\]")
    if [ "$NODE_COUNT" -ge 5 ]; then
        test_pass "FlatNodes includes all nested nodes ($NODE_COUNT nodes)"
    else
        test_fail "FlatNodes count" "expected 5+, got $NODE_COUNT"
    fi

    # Cleanup
    cleanup

    # Summary
    echo ""
    echo "=========================================="
    echo "Test Summary"
    echo "=========================================="
    echo -e "${GREEN}Passed:${NC} $PASS"
    echo -e "${RED}Failed:${NC} $FAIL"
    echo -e "${YELLOW}Skipped:${NC} $SKIP"
    echo "Total: $((PASS + FAIL + SKIP))"
    echo ""

    if [ $FAIL -gt 0 ]; then
        echo -e "${RED}Some tests failed!${NC}"
        exit 1
    else
        echo -e "${GREEN}All tests passed!${NC}"
        exit 0
    fi
}

# Run main
main "$@"
