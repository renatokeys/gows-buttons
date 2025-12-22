#!/bin/bash
set -e

# Copy buttons.go to src/server/
cp patches/buttons.go src/server/buttons.go

# Add SendButtons to proto if not exists
if ! grep -q "rpc SendButtons" proto/gows.proto; then
  # Add SendButtons RPC after SendButtonReply
  sed -i 's/rpc SendButtonReply (ButtonReplyRequest) returns (MessageResponse);/rpc SendButtonReply (ButtonReplyRequest) returns (MessageResponse);\n  rpc SendButtons (SendButtonsRequest) returns (MessageResponse);/' proto/gows.proto
fi

# Add Button types to proto if not exists
if ! grep -q "enum ButtonType" proto/gows.proto; then
  cat >> proto/gows.proto << 'EOF'

//
// Buttons
//
enum ButtonType {
  BUTTON_REPLY = 0;
  BUTTON_URL = 1;
  BUTTON_CALL = 2;
  BUTTON_COPY = 3;
}

message Button {
  ButtonType type = 1;
  string text = 2;
  string id = 3;
  string url = 4;
  string phoneNumber = 5;
  string copyCode = 6;
}

message SendButtonsRequest {
  Session session = 1;
  string jid = 2;
  string header = 3;
  bytes headerImage = 4;
  string body = 5;
  string footer = 6;
  repeated Button buttons = 7;
}
EOF
fi

echo "Buttons patch applied successfully!"
