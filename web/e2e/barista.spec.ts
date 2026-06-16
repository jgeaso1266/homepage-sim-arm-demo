import { expect, test } from '@playwright/test'

test.describe('barista arm-swap demo', () => {
	test('renders the scene, plays the brew, and swaps arms', async ({ page }) => {
		await page.goto('/')

		// The caption renders and the 3D canvas mounts.
		await expect(page.getByText('Same motion code. Different arm.')).toBeVisible()
		await expect(page.locator('canvas')).toBeAttached({ timeout: 30_000 })

		// The xArm6 trajectory loads (button enabled), then "Make coffee" starts playback.
		const brew = page.getByRole('button', { name: /make coffee/i })
		await expect(brew).toBeEnabled({ timeout: 30_000 })
		await brew.click()
		await expect(page.getByRole('button', { name: /brewing/i })).toBeVisible()

		// The code drawer shows the planning code with the arm model highlighted.
		await page.getByRole('button', { name: /show the code/i }).click()
		await expect(page.locator('pre code')).toContainText('PlanMotion')
		await expect(page.locator('.hl').first()).toHaveText('xarm6')

		// Switching arms loads the other asset and updates the highlighted model.
		const ur5eAsset = page.waitForResponse(/ur5e\.brew\.json/)
		await page.getByRole('button', { name: 'UR5e', exact: true }).click()
		await ur5eAsset
		await expect(page.locator('.hl').first()).toHaveText('ur5e')
	})
})
