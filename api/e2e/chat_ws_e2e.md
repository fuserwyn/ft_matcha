# Chat WS E2E Test Cases

## Preconditions

- API is running.
- Two users exist: `A` and `B`.
- Both have valid JWT tokens.

---

## Case 1: Match flow (2 users)

1. User `A` likes `B`: `POST /api/v1/users/{B}/like`.
2. User `B` likes `A`: `POST /api/v1/users/{A}/like`.
3. Verify both see each other in `GET /api/v1/matches`.
4. Open WS for both:
   - `ws://localhost:8080/api/v1/ws/chat?token=<tokenA>`
   - `ws://localhost:8080/api/v1/ws/chat?token=<tokenB>`
5. From `A`, send WS payload:
   ```json
   {"to_user_id":"<B_UUID>","content":"hello from A"}
   ```
6. Expected:
   - `A` receives `type=message`.
   - `B` receives `type=message`.
   - Message is persisted (`GET /api/v1/users/{B}/messages` from `A`).

---

## Case 2: No-match blocked flow

1. Create users `C` and `D` with only one-way like (`C -> D`).
2. Open WS as `C`.
3. Send:
   ```json
   {"to_user_id":"<D_UUID>","content":"should fail"}
   ```
4. Expected:
   - Receive `type=error` with `"can only message matches"`.
   - No row inserted in `messages`.

---

## Case 3: Heartbeat / reconnect

1. Open WS as `A`.
2. Keep connection idle for > 60s.
3. Expected:
   - Server sends periodic ping (every ~25s).
   - Connection remains alive if pong is returned by client.
4. Force disconnect (close socket/network).
5. Reconnect with same token.
6. Send new message and verify normal delivery/persistence.

---

## Case 4: Rate limit / anti-spam

1. Open WS as `A` (matched with `B`).
2. Send >20 messages within 10 seconds.
3. Expected:
   - First messages are accepted.
   - Excess messages receive `type=error` with rate-limit message.
