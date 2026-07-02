Write a workflow for reviewing and enforcing every pattern and convention file in `/docs`.

Start by splitting the work into individual convention-review tasks:

1. Read every pattern and convention file in `/docs`.
2. For each file, create an individual task file that contains:

   * The source convention file path
   * A concise summary of the convention
   * The specific codebase areas likely affected by that convention
   * The review objective
   * The expected remediation criteria
3. Pass these individual task files into the workflow.

For each convention-review task:

Step 1: Review the codebase for violations of the convention.

* Identify every place where the code does not conform to the documented pattern or convention.
* Fix the issues discovered.
* Preserve existing behavior unless the convention explicitly requires a behavior change.
* Do not use any git commands.
* Do not use build commands.
* Do not use broad test commands.
* Avoid actions that could interfere with another Claude working in the same branch.

Step 2: Use 2 adversarial review agents to refute the convention fix.

Each adversarial review agent should independently inspect the changes and attempt to find flaws, including:

* Missed convention violations
* Incorrect interpretation of the convention
* Over-applied or under-applied changes
* Behavior regressions
* Inconsistencies with neighboring code
* Style or architecture drift
* Any fixes that conflict with other conventions in `/docs`

The adversarial agents must not use any git commands, build commands, or broad test commands.

After all convention-review tasks are complete:

Step 3: Apply and reconcile all fixes.

* Combine the fixes across all convention-review tasks.
* Resolve conflicts between conventions by following the more specific convention first, then the broader convention.
* Re-check affected code paths for consistency.
* Run the build.
* Run the relevant tests.
* Fix any failures.
* Once the build and relevant tests pass, commit the changes.
* Create a PR with a summary that includes:

  * Which `/docs` convention files were reviewed
  * What categories of violations were found
  * What changes were made
  * What tests/build commands were run
  * Any conventions that were ambiguous or required judgment calls
