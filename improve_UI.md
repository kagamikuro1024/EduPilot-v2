```text
You are a senior full-stack engineer working on an existing AI Teaching Assistant application.

Your task is to design and implement production-ready “My Profile” and “Settings” features, primarily for STUDENT users, while preserving all existing application behavior and AI functionality.

Do not modify or refactor the AI core as part of this task.

## Project context

The application currently contains:

- Next.js frontend using TypeScript, React, Tailwind CSS, Zustand, Axios, and the App Router.
- Spring Boot Java backend using Spring Security, JWT, JPA, PostgreSQL, and BCrypt.
- Python FastAPI/gRPC AI service.
- PostgreSQL and Redis.
- A Docker-based Hugging Face Space deployment.
- Existing roles:
  - STUDENT
  - TA
  - ADMIN
- Existing JWT authentication and an authenticated Axios client named `javaClient`.
- A shared workspace layout and navigation system.
- Existing persisted authentication state in `frontend/src/store/authStore.ts`.
- Existing database migrations under `db/migration/`.
- Hugging Face incremental migrations loaded through:
  - `db/00-init.sql`
  - `db/hf-incremental.sql`

Inspect the repository before changing anything. Do not assume that file names or implementations are unchanged.

## Main objective

Create two new user-facing areas:

1. `/profile` — “My Profile”
2. `/settings` — “Settings”

Students must be able to:

- View their account information.
- Update their display name.
- Upload, replace, and remove their avatar.
- Change their password securely.
- Choose light, dark, or system theme.
- Adjust basic accessibility preferences.
- Select their preferred default page after login.
- Reset appearance preferences.
- Log out safely.

The implementation must use real backend data. Do not use mock profile information.

## Non-negotiable constraints

Do not:

1. Change existing AI behavior, prompts, RAG logic, agent orchestration, tools, embeddings, citations, gRPC services, or Python AI code.
2. Change existing chat, forum, assignment, analytics, or document-processing logic.
3. Change existing authentication endpoint contracts.
4. Change existing role permissions.
5. Allow users to modify their email, role, internal user ID, or student code through profile editing.
6. Store plaintext passwords.
7. Put passwords, JWTs, API keys, or sensitive information into logs.
8. Expose local filesystem paths through API responses.
9. Trust uploaded file names or extensions without validating their content.
10. introduce mock profile data.
11. Break the Hugging Face all-in-one Docker deployment.
12. Overwrite unrelated working-tree changes.
13. Perform a broad UI redesign unrelated to Profile and Settings.
14. Add a new external storage service or paid dependency.

If a requirement would affect application logic outside this scope, document it instead of implementing it.

## Required frontend experience

### 1. My Profile page

Create a responsive page at:

```text
/profile
```

Use the existing application shell and `WorkspaceLayout`.

The page should contain:

#### Profile header

Display:

- User avatar.
- Full name.
- Email.
- Student code, if available.
- Role.
- Account creation date, if available.
- A small profile-completion indicator.

The profile-completion indicator may consider:

- Avatar configured.
- Full name configured.
- Student code available.
- Email available.

Do not create fake completion criteria requiring fields the application does not support.

#### Editable profile form

Allow users to edit only safe profile fields:

- Full name.

The following must be read-only:

- Email.
- Role.
- Student code.
- Account ID.
- Account creation date.

Requirements:

- Validate the full name.
- Trim leading and trailing whitespace.
- Prevent empty names.
- Use clear inline validation messages.
- Disable the submit button when nothing changed.
- Show loading state while saving.
- Show success and error notifications.
- Update `authStore.fullName` after a successful change so the navigation and profile header update immediately.

#### Avatar management

Allow users to:

- Upload an avatar.
- Replace the current avatar.
- Remove the avatar.
- Preview the selected image before uploading.
- Cancel the pending selection.

Supported formats:

- JPEG
- PNG
- WebP

Validation:

- Validate both MIME type and decoded image content.
- Maximum size: 2 MB.
- Reject SVG and animated formats.
- Do not trust the original file name.
- Generate an opaque server-side filename if filesystem storage is used.
- Prevent path traversal.
- Remove or replace the old avatar safely.
- Show initials as a fallback when no avatar exists.
- Revoke browser object URLs when no longer needed.

The avatar must appear consistently in:

- My Profile.
- Workspace navigation/header.
- Any existing user menu that currently shows initials.

Create a reusable authenticated avatar component instead of duplicating avatar-fetching logic.

### 2. Settings page

Create a responsive page at:

```text
/settings
```

Organize it into clear sections.

#### Appearance

Support:

- Light theme.
- Dark theme.
- System theme.

Requirements:

- Apply the selected theme immediately.
- Avoid a flash of the wrong theme during page load.
- Respect the operating system theme when “System” is selected.
- Persist the preference.
- Maintain readable contrast in all existing pages.
- Do not redesign every component unless a small compatibility change is required.
- Use the project’s existing Tailwind approach.

#### Accessibility

Add minimal useful accessibility preferences:

- Reduced motion: on/off.
- Interface font size:
  - Small
  - Default
  - Large

Requirements:

- Reduced motion must disable or minimize nonessential animations.
- Continue respecting the system-level `prefers-reduced-motion` setting.
- Font-size changes must not break page layouts.
- Preferences should apply globally through CSS classes or variables.
- Provide a “Reset appearance settings” action.

#### Default student landing page

Allow STUDENT users to choose their preferred page after login:

- Assignments
- Chat

Requirements:

- Preserve existing TA and ADMIN post-login redirects.
- Only apply this preference to STUDENT users.
- If the selected route is unavailable, fall back to the current default redirect.
- Do not create redirect loops.

#### Account section

Include:

- A link to My Profile.
- A “Change password” action.
- A “Log out” action.
- Brief account-security guidance.

Do not implement account deletion unless explicitly requested later.

## Password-change feature

Add a secure password-change form.

Recommended endpoint:

```text
PATCH /api/v1/users/me/password
```

Request:

```json
{
  "current_password": "current password",
  "new_password": "new password"
}
```

Successful response:

```text
204 No Content
```

Requirements:

1. Require the current password.
2. Verify it using the existing password encoder.
3. Reject an incorrect current password without exposing sensitive details.
4. Require password confirmation in the frontend.
5. Never send the confirmation field to the backend.
6. Reuse the existing registration password policy if one exists.
7. If no policy exists, enforce:
   - Minimum 8 characters.
   - Maximum 72 characters for BCrypt compatibility.
   - At least one uppercase letter.
   - At least one lowercase letter.
   - At least one digit.
8. Prevent the new password from matching the current password.
9. Store only the BCrypt hash.
10. Do not log either password.
11. Return safe, structured validation errors.
12. Clear password fields after completion or failure where appropriate.
13. Log the user out after a successful password change and redirect to `/login`.

Where practical, invalidate JWTs issued before the password change using a backward-compatible mechanism such as `password_changed_at` or a token version.

Do not change the existing login/register response contract.

Add tests proving that:

- The old password stops working.
- The new password works.
- Incorrect current passwords are rejected.
- Another user cannot change the current user’s password.
- Password values do not appear in logs or responses.

## Backend profile API

Use authenticated “current user” endpoints rather than accepting arbitrary user IDs.

Recommended endpoints:

```text
GET    /api/v1/users/me
PATCH  /api/v1/users/me
GET    /api/v1/users/me/avatar
POST   /api/v1/users/me/avatar
DELETE /api/v1/users/me/avatar
PATCH  /api/v1/users/me/password
GET    /api/v1/users/me/preferences
PATCH  /api/v1/users/me/preferences
```

The exact endpoint organization may be adapted to existing project conventions, but it must remain RESTful and self-scoped.

Example profile response:

```json
{
  "id": "uuid",
  "full_name": "Nguyen Van A",
  "email": "student@example.com",
  "student_code": "20225104",
  "role": "STUDENT",
  "avatar_available": true,
  "created_at": "2026-07-05T10:30:00Z"
}
```

Do not return:

- Password hashes.
- Internal storage paths.
- JWT secrets.
- Other users’ information.
- Security implementation details.

Example profile update:

```json
{
  "full_name": "Nguyen Van A"
}
```

Reject attempts to update fields such as:

- `role`
- `email`
- `student_code`
- `password`
- `id`

Prefer strict request DTOs so unknown or forbidden fields are not silently accepted.

## Preference persistence

Persist user preferences across devices.

Suggested preferences:

```json
{
  "theme": "SYSTEM",
  "font_size": "DEFAULT",
  "reduce_motion": false,
  "default_student_page": "ASSIGNMENTS"
}
```

Allowed values:

```text
theme:
- LIGHT
- DARK
- SYSTEM

font_size:
- SMALL
- DEFAULT
- LARGE

default_student_page:
- ASSIGNMENTS
- CHAT
```

Use explicit validated values rather than arbitrary strings.

Use local storage as a fast frontend cache, but treat the authenticated backend preference as the source of truth after login.

Recommended synchronization behavior:

1. Apply locally cached appearance settings immediately.
2. After authentication, fetch backend preferences.
3. Apply backend preferences.
4. Update both backend and local cache when the user changes a setting.
5. Use safe defaults if no preference row exists.
6. Do not block the whole application while preferences load.

## Database changes

Add the smallest necessary migration.

A reasonable design is:

### User profile additions

Add safe user-profile metadata if missing:

- Avatar storage reference or avatar metadata.
- Password-changed timestamp if used for token invalidation.

### User preferences

Create a dedicated table such as:

```text
user_preferences
```

Possible columns:

- `user_id` UUID primary key and foreign key.
- `theme`
- `font_size`
- `reduce_motion`
- `default_student_page`
- `updated_at`

Use appropriate constraints and defaults.

Requirements:

- Follow the next available migration version; do not assume a version number without inspecting the directory.
- Make incremental Hugging Face migrations idempotent.
- Update both:
  - `db/00-init.sql`
  - `db/hf-incremental.sql`
- Existing users must receive safe defaults.
- Do not delete or reset existing user data.
- Do not modify unrelated tables.

## Avatar storage decision

Choose one implementation after inspecting the current deployment.

Preferred options:

### Option A: Persistent upload directory

Store avatars under a controlled path beneath:

```text
HF_DATA_DIR/uploads/avatars
```

Persist only an opaque storage identifier in PostgreSQL.

### Option B: Database binary storage

Use PostgreSQL binary storage only if the project’s deployment constraints make filesystem storage unreliable.

Whichever option is selected:

- Document the tradeoff.
- Enforce the 2 MB limit.
- Validate actual image content.
- Set correct response `Content-Type`.
- Add cache headers where safe.
- Do not expose local paths.
- Delete replaced files only after the new upload succeeds.
- Prevent orphaned files where practical.

## Navigation integration

Add access to the new pages through the existing navigation system.

Requirements:

- Add “My Profile”.
- Add “Settings”.
- Use appropriate existing icons.
- Maintain role-aware navigation.
- Do not duplicate navigation entries.
- Update the user/avatar menu if one exists.
- Keep mobile navigation usable.
- Ensure active-route highlighting works.

## UI and accessibility requirements

The pages must:

- Match the existing visual language.
- Be responsive.
- Be keyboard accessible.
- Use semantic labels.
- Provide visible focus states.
- Use accessible modal/dialog behavior.
- Avoid color-only status communication.
- Have sufficient contrast.
- Provide loading skeletons or compact loading indicators.
- Handle empty/error states.
- Show confirmations for destructive actions such as avatar removal or preference reset.
- Avoid excessive animation.

Use the project’s existing notification/toast library.

## Security requirements

- Every profile endpoint must require authentication.
- Users may only access their own profile and preferences.
- Never accept a user ID from the frontend for self-service operations.
- Validate all request payloads server-side.
- Validate uploaded files server-side.
- Apply upload-size limits before reading large files into memory.
- Never expose password hashes.
- Never log password fields.
- Do not leak whether another account exists.
- Preserve existing Spring Security behavior.
- Prevent mass-assignment vulnerabilities.
- Add appropriate cache-control headers to sensitive profile responses.
- Do not put JWTs in URLs.
- Do not weaken CORS or CSRF configuration.
- Keep avatar handling resistant to path traversal and malformed-image attacks.

## Error handling

Return predictable errors for:

- Invalid full name.
- Incorrect current password.
- Weak new password.
- Unsupported avatar type.
- Avatar exceeding the size limit.
- Malformed image.
- Missing avatar.
- Invalid preference value.
- Unauthenticated request.
- Internal storage failure.

Frontend errors must be understandable to users and must not expose stack traces or internal paths.

## Testing requirements

### Backend tests

Add tests for:

- Fetching the authenticated user profile.
- Updating the full name.
- Rejecting empty names.
- Preventing updates to protected fields.
- Fetching and updating preferences.
- Preference defaults.
- Uploading a valid avatar.
- Rejecting oversized avatars.
- Rejecting invalid MIME types.
- Rejecting malformed images.
- Removing an avatar.
- Avatar ownership.
- Successful password change.
- Incorrect current password.
- Weak password rejection.
- JWT invalidation behavior after password change.
- Unauthenticated requests.
- Cross-user access attempts.

### Frontend tests

Add tests for:

- Profile loading.
- Profile editing.
- Store update after name change.
- Avatar preview.
- Avatar validation.
- Avatar removal confirmation.
- Password confirmation mismatch.
- Theme switching.
- System-theme behavior.
- Reduced-motion setting.
- Font-size setting.
- Preference reset.
- Student landing-page preference.
- TA/ADMIN redirect behavior remaining unchanged.
- Loading and API error states.

Mock network requests. Do not require real credentials or external services.

## Verification

Run the project’s relevant commands, including where available:

```text
Frontend lint
Frontend tests
Frontend production build
Java tests
Java production build
Docker image build
Hugging Face smoke test
```

Verify manually or through automated tests:

1. Login as STUDENT.
2. Open My Profile.
3. Change full name.
4. Upload an avatar.
5. Refresh and confirm persistence.
6. Open Settings.
7. Change theme.
8. Refresh and confirm persistence.
9. Change font size and reduced-motion preference.
10. Change default landing page.
11. Log out and log in again.
12. Confirm the selected student landing page is used.
13. Change password.
14. Confirm logout occurs.
15. Confirm the old password fails.
16. Confirm the new password works.
17. Confirm TA and ADMIN flows still work.
18. Confirm chat, assignments, analytics, forum, and AI endpoints are unaffected.

## Required implementation process

### Phase 1: Inspect

Before editing:

- Inspect the current User entity.
- Inspect authentication and JWT generation.
- Inspect `authStore`.
- Inspect `javaClient`.
- Inspect navigation and workspace layout.
- Inspect post-login redirect logic.
- Inspect migrations.
- Inspect Hugging Face persistent storage paths.
- Inspect existing theme support.

Produce a short implementation plan based on actual repository evidence.

### Phase 2: Implement backend

Implement:

- Profile DTOs.
- Preferences DTOs.
- Self-scoped controller.
- Profile service.
- Preference persistence.
- Secure password change.
- Avatar storage.
- Necessary migrations.
- Backend tests.

### Phase 3: Implement frontend

Implement:

- Profile service/client.
- Preference store or hook.
- Theme bootstrap.
- My Profile page.
- Settings page.
- Avatar component.
- Password dialog/form.
- Navigation integration.
- Frontend tests.

### Phase 4: Verify

Run all relevant tests and builds.

Test the Docker Space locally when possible.

### Phase 5: Report

Provide:

1. Files changed.
2. Database changes.
3. API endpoints added.
4. Security decisions.
5. Avatar-storage decision.
6. Tests added.
7. Commands executed.
8. Verification results.
9. Known limitations.
10. Any recommendations intentionally not implemented.

## Definition of done

The task is complete only when:

- `/profile` works with real authenticated user data.
- `/settings` works and persists preferences.
- Full-name updates appear immediately across the UI.
- Avatar upload, replacement, loading, and removal work.
- Password change securely verifies the current password.
- A successful password change logs the user out.
- Theme supports Light, Dark, and System.
- Accessibility preferences apply globally.
- STUDENT default-page preference works.
- TA and ADMIN redirects remain unchanged.
- Existing AI, chat, assignments, analytics, forum, and document features still work.
- All relevant tests and builds pass.
- The Docker Space remains compatible with Hugging Face CPU Basic.
- No unrelated files are modified.
- No mock data remains in the implementation.

Begin by inspecting the repository and presenting a focused implementation plan. Then implement and verify the feature end-to-end.

## Mandatory design-system consistency

Visual consistency with the existing application is a non-negotiable requirement.

The new Profile and Settings pages must look like native parts of the current product. Do not create a new visual style, standalone dashboard design, or separate design system.

Before implementing any UI:

1. Inspect representative existing pages, including:
   - Assignments
   - Chat
   - Analytics
   - Documents
   - Login/Register
2. Inspect the existing:
   - `WorkspaceLayout`
   - Navigation components
   - Page headers
   - Cards
   - Buttons
   - Form fields
   - Dialogs and modals
   - Dropdowns
   - Tabs
   - Toast notifications
   - Loading states
   - Empty states
   - Avatar or user-menu components
3. Identify the application’s current:
   - Color palette
   - Typography scale
   - Font weights
   - Border radii
   - Shadows
   - Spacing rhythm
   - Container widths
   - Grid behavior
   - Icon style
   - Button hierarchy
   - Form-validation style
   - Responsive breakpoints
   - Dark-mode conventions

Reuse existing components, CSS variables, utility classes, design tokens, and layout patterns wherever possible.

Do not:

- Introduce a new color palette.
- Introduce a new font family.
- Create different button, card, input, modal, or navigation styles.
- Add gradients, glassmorphism, excessive shadows, or decorative effects unless already used by the product.
- Add a new UI framework or component library.
- Duplicate an existing shared component.
- Create one-off Tailwind styles when an equivalent reusable component already exists.
- Modify shared components in a way that visually breaks existing pages.
- Redesign the global navigation or workspace shell.
- Make Profile and Settings feel like a separate application.

When no reusable component exists, create the smallest reusable component that follows the nearest existing pattern.

The pages must use:

- The existing `WorkspaceLayout`.
- The existing page-width and content-padding conventions.
- The existing heading hierarchy.
- The existing card and section styles.
- The existing primary, secondary, ghost, and destructive button styles.
- The existing input, label, validation, and helper-text styles.
- The existing icon library and icon sizing.
- The existing toast and confirmation-dialog patterns.
- The existing loading and error-state conventions.

Do not “improve” or reinterpret the visual design beyond what is necessary for the requested features.

## Visual verification requirement

After implementation, perform visual verification for:

- Desktop width.
- Tablet width.
- Mobile width.
- Light theme.
- Dark theme.
- Loading states.
- Validation errors.
- Empty avatar state.
- Uploaded avatar state.
- Long user names and email addresses.

Compare the new pages directly with existing product pages.

The implementation is not complete if Profile or Settings appears to use a different design language from the rest of the application.

In the final report, explicitly list:

1. Existing components and patterns reused.
2. Any new shared UI components created.
3. Why each new component was necessary.
4. Visual consistency checks performed.

```