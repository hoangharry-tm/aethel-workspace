-- Migration 08 UP: Unified event log for the document timeline (US-14).
-- Every significant state change — routing, delivery, escalation, etc. —
-- writes a row here. The frontend timeline view reads from this table.
--
-- event_type values (immutable after insert, never updated):
--   CREATED            — document first logged by Reception
--   ROUTING_APPLIED    — auto-routing rule matched and applied
--   ROUTING_MANUAL_OVERRIDE — Reception overrode the suggestion (US-04)
--   STOP_NOTIFIED      — recipient at a routing stop notified
--   STOP_CONFIRMED     — recipient confirmed receipt at stop
--   STOP_REJECTED      — recipient rejected at stop (halts chain)
--   STOP_SKIPPED       — stop bypassed by admin
--   HANDOFF_ATTEMPTED  — Reception tried physical handoff (US-07)
--   DELIVERED          — document marked delivered
--   ACKNOWLEDGED       — recipient digitally acknowledged (US-08)
--   ESCALATED          — SLA breach; escalation triggered (US-12)
--   ESCALATION_CLEARED — document acknowledged after escalation
--   STATUS_CHANGED     — any other status transition

CREATE TABLE {{ .Schema }}.{{ T "dispatch_events" }} (
    id                   uuid        NOT NULL DEFAULT gen_random_uuid(),
    dispatch_id          uuid        NOT NULL,
    -- No FK on routing_rule_id: event history must survive rule deletion.
    -- Matches the same intentional pattern as audit_ledger.organization_id.
    routing_rule_id      uuid,
    routing_stop_order   smallint,
    event_type           varchar(50) NOT NULL,
    actor_user_id        uuid,
    target_user_id       uuid,
    target_department_id uuid,
    stop_status          varchar(20)
        CONSTRAINT {{ T "dispatch_events" }}_stop_status_ck
            CHECK (stop_status IN ('PENDING', 'CONFIRMED', 'REJECTED', 'SKIPPED')),
    note                 text,
    metadata             jsonb,
    created_at           timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT {{ T "dispatch_events" }}_pkey        PRIMARY KEY (id),
    CONSTRAINT {{ T "dispatch_events" }}_dispatch_fk FOREIGN KEY (dispatch_id)
        REFERENCES {{ .Schema }}.{{ T "dispatches" }} (id) ON DELETE CASCADE,
    CONSTRAINT {{ T "dispatch_events" }}_actor_fk    FOREIGN KEY (actor_user_id)
        REFERENCES {{ .Schema }}.{{ T "users" }} (id) ON DELETE RESTRICT,
    CONSTRAINT {{ T "dispatch_events" }}_target_fk   FOREIGN KEY (target_user_id)
        REFERENCES {{ .Schema }}.{{ T "users" }} (id) ON DELETE RESTRICT,
    CONSTRAINT {{ T "dispatch_events" }}_dept_fk     FOREIGN KEY (target_department_id)
        REFERENCES {{ .Schema }}.{{ T "departments" }} (id) ON DELETE RESTRICT
);

-- Timeline view: all events for one dispatch, chronological (US-14)
CREATE INDEX {{ T "dispatch_events" }}_dispatch_time_idx
    ON {{ .Schema }}.{{ T "dispatch_events" }} (dispatch_id, created_at ASC);

-- Pending-stop lookup: find which stops are still awaiting confirmation
CREATE INDEX {{ T "dispatch_events" }}_pending_stops_idx
    ON {{ .Schema }}.{{ T "dispatch_events" }} (dispatch_id, routing_stop_order)
    WHERE stop_status = 'PENDING';
