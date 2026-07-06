## 0.1.0 (Unreleased)

ENHANCEMENTS:

* resource/mxroute_email_account: `password_wo` is now optional, so an existing
  mailbox no longer needs a password in its configuration. It is still required
  when **creating** a mailbox (enforced with a clear error, matching the API,
  which requires a password on create but not on update), and bumping
  `password_wo_version` without supplying a password is now rejected rather than
  silently setting an empty password.

FEATURES:
