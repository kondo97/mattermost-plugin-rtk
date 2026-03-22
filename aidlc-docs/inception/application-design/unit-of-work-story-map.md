# Unit of Work — Story Map

## Story-to-Unit Assignment

| Story | Title | Primary Unit | Supporting Unit(s) |
|---|---|---|---|
| US-001 | Configure Cloudflare Credentials | Unit 5 | Unit 2 (config status API) |
| US-002 | Toggle Feature Flags | Unit 5 | Unit 2 (feature flags in API responses) |
| US-003 | See Call Button in Channel Header | Unit 3 | Unit 2 (config status API for credential check) |
| US-004 | Call Button States Reflect Call Status | Unit 3 | Unit 1 (call state), Unit 2 (WS events) |
| US-005 | Start a Call | Unit 1 | Unit 2 (POST /calls endpoint), Unit 3 (UI dispatch) |
| US-006 | Floating Widget Appears When Call Starts or Joined | Unit 3 (FloatingWidget) | Unit 4 (Call Page opened from widget) |
| US-007 | Custom Post Appears When Call Starts | Unit 4 | Unit 2 (custom_cf_call_started WS event) |
| US-008 | Channel Call Toast Bar Appears | Unit 3 | Unit 2 (WS events) |
| US-009 | Join a Call from the Custom Post | Unit 1 | Unit 2 (POST /calls/{id}/token), Unit 4 (Call Page) |
| US-010 | Custom Post Disables "Join" Button When Already in Call | Unit 4 | Unit 3 (Redux myActiveCall selector) |
| US-011 | Channel Header Shows "In Call" While User Is in a Call | Unit 3 | Unit 2 (WS events) |
| US-012 | Custom Post Updates in Real-Time via WebSocket | Unit 3 (WS handlers) | Unit 4 (CallPost re-render) |
| US-013 | Leave a Call by Closing the Tab | Unit 4 (sendBeacon) | Unit 1 (LeaveCall logic), Unit 2 (POST /calls/{id}/leave) |
| US-015 | Host Ends the Call for All Participants | Unit 1 | Unit 2 (DELETE /calls/{id}), Unit 4 (end call trigger from call page) |
| US-016 | Post Card Switches to "Ended" State When Call Ends | Unit 4 | Unit 2 (custom_cf_call_ended WS event), Unit 3 (Redux update) |
| US-018 | Receive Incoming Call Push Notification | Unit 6 | Unit 1 (CreateCall triggers push) |
| US-019 | Join a Call from Push Notification | Unit 6 | Unit 2 (POST /calls/{id}/token response includes feature flags) |
| US-020 | Dismiss Incoming Call Notification | Unit 2 (POST /calls/{id}/dismiss) | Unit 3 (custom_cf_notification_dismissed WS handler) |
| US-021 | Web Worker Script is Served by the Plugin | Unit 2 | — |
| US-022 | Active Call Blocking — Cannot Start a Second Call | Unit 1 (KVStore check) | Unit 2 (error response), Unit 3 (error modal) |
| US-023 | Switch Call Modal — Joining a Different Call While Already in One | Unit 3 | Unit 1 (LeaveCall + JoinCall sequence) |
| US-024 | DM/GM Ringing — Incoming Call Notification with Ignore and Join | Unit 3 | Unit 2 (custom_cf_call_started WS event) |
| US-025 | Last-Participant Auto-End — Call Ends When All Leave | Unit 1 | Unit 2 (WS event emission), Unit 4 (call post update) |

## Per-Unit Story Summary

### Unit 1: RTK Integration
**Primary**: US-005, US-009, US-013, US-015, US-022, US-025
**Supporting**: US-006, US-018, US-019, US-023, US-024
**Journey coverage**: Initiating (J2), Joining (J4), Leaving (J6), Ending (J7), Edge Cases

### Unit 2: Server API & WebSocket
**Primary**: US-020, US-021
**Supporting**: US-001, US-002, US-003, US-004, US-005, US-007, US-008, US-009, US-011, US-012, US-013, US-015, US-016, US-018, US-019, US-022, US-023, US-024, US-025
**Journey coverage**: All journeys (API layer for every feature)

### Unit 3: Webapp - Channel UI
**Primary**: US-003, US-004, US-006, US-008, US-011, US-012, US-023, US-024
**Supporting**: US-005, US-009, US-010, US-013, US-016, US-020, US-022
**Journey coverage**: Admin Setup (J1 partial), Initiating (J2), Notifications (J3), In-Call State (J5), Edge Cases

### Unit 4: Webapp - Call Page & Post
**Primary**: US-007, US-010, US-013, US-016
**Supporting**: US-006, US-009, US-012, US-015, US-025
**Journey coverage**: Notifications (J3), Joining (J4 partial), In-Call State (J5), Leaving (J6), Ending (J7)

### Unit 5: Admin & Config
**Primary**: US-001, US-002
**Supporting**: US-003
**Journey coverage**: Admin Setup (J1)

### Unit 6: Mobile Support
**Primary**: US-018, US-019
**Supporting**: US-020
**Journey coverage**: Mobile Incoming Call (J8)

## Coverage Summary

| Total Stories | Assigned | Unassigned |
|---|---|---|
| 23 | 23 | 0 |

All stories from US-001 through US-025 (excluding US-014 which does not exist in the stories document) are assigned to at least one unit.
