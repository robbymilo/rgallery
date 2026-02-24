import { chromium } from 'playwright';

(async () => {
  const browser = await chromium.launch();
  const context = await browser.newContext({
    viewport: { width: 1600, height: 6000 },
  });

  await context.addCookies([
    {
      name: 'session',
      value: '46e3ef0c-5a4a-4634-b440-1883b83140b4',
      domain: 'demo-internal.rgallery.app',
      path: '/',
      httpOnly: true,
      secure: true,
      sameSite: 'Lax',
    },
  ]);

  const page = await context.newPage();
  await page.goto('https://demo-internal.rgallery.app/', {
    waitUntil: 'networkidle',
  });

  await page.addStyleTag({
    content: `
    #root .bg-charcoal-900 > .absolute {
      top: 500px !important;
      left: 105px !important;
      transform: none !important;
    }
  `,
  });

  const selector = '#root .bg-charcoal-900 > .absolute';

  // hover it
  await page.hover(selector);

  // give hover animations / transitions time
  await page.waitForTimeout(200);

  // screenshot
  await page.screenshot({ path: 'screenshot.png' });

  await browser.close();
})();
