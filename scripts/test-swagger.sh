#!/bin/bash

# Colors
CYAN='\033[0;36m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${CYAN}=========================================${NC}"
echo -e "${CYAN}    SWAGGER UI VERIFICATION SCRIPT       ${NC}"
echo -e "${CYAN}=========================================${NC}"
echo ""

# 1. Check if server is running (Usamos -f para que falle si hay un 404 y la ruta correcta)
echo -e "${YELLOW}[1/4]${NC} Checking if server is running..."
if curl -s -f http://localhost:8080/api/v1/health > /dev/null; then
    echo -e "${GREEN}  ✓ Server is running and responding${NC}"
else
    echo -e "${RED}  ✗ Server is not running or returning errors${NC}"
    echo -e "${YELLOW}  Asegúrate de ejecutar 'make run' en otra terminal${NC}"
    exit 1
fi

# 2. Check Swagger JSON (Evitamos la dependencia de 'jq' usando 'grep' nativo)
echo ""
echo -e "${YELLOW}[2/4]${NC} Checking Swagger JSON..."
if curl -s -f http://localhost:8080/swagger/doc.json | grep -q "swagger"; then
    echo -e "${GREEN}  ✓ Swagger JSON is accessible and valid${NC}"
else
    echo -e "${RED}  ✗ Swagger JSON not found at /swagger/doc.json${NC}"
    echo -e "${YELLOW}  Asegúrate de haber ejecutado 'make swagger' (o swagger-init)${NC}"
    exit 1
fi

# 3. Verify Swagger UI accessibility
echo ""
echo -e "${YELLOW}[3/4]${NC} Verifying Swagger UI HTML..."
if curl -s -f http://localhost:8080/swagger/index.html | grep -q "Swagger UI"; then
    echo -e "${GREEN}  ✓ Swagger UI interface is accessible${NC}"
else
    echo -e "${RED}  ✗ Swagger UI not accessible${NC}"
    exit 1
fi

# 4. Check for 'jq' installation (Optional enhancement)
echo ""
echo -e "${YELLOW}[4/4]${NC} Environment Check..."
if command -v jq &> /dev/null; then
    ENDPOINT_COUNT=$(curl -s http://localhost:8080/swagger/doc.json | jq '.paths | length')
    echo -e "${GREEN}  ✓ jq is installed. Documented endpoints: ${ENDPOINT_COUNT}${NC}"
else
    echo -e "${CYAN}  ℹ 'jq' is not installed (expected on Windows). Skipping JSON deep analysis.${NC}"
fi

echo ""
echo -e "${CYAN}=========================================${NC}"
echo -e "${GREEN}✓ All critical checks passed!${NC}"
echo ""
echo -e "${YELLOW}Access Swagger UI at:${NC}"
echo -e "${GREEN}  http://localhost:8080/swagger/index.html${NC}"
echo -e "${CYAN}=========================================${NC}"
