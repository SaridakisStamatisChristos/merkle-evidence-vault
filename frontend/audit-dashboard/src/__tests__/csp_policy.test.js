import { describe, expect, it } from 'vitest';
import { DASHBOARD_CSP, DASHBOARD_SECURITY_HEADERS } from '../cspPolicy';

describe('dashboard CSP policy', () => {
  it('includes key anti-xss directives', () => {
    expect(DASHBOARD_CSP).toContain("default-src 'self'");
    expect(DASHBOARD_CSP).toContain("script-src 'self'");
    expect(DASHBOARD_CSP).toContain("object-src 'none'");
    expect(DASHBOARD_CSP).toContain("frame-ancestors 'none'");
  });

  it('exports baseline hardening headers', () => {
    expect(DASHBOARD_SECURITY_HEADERS['Content-Security-Policy']).toBe(DASHBOARD_CSP);
    expect(DASHBOARD_SECURITY_HEADERS['X-Content-Type-Options']).toBe('nosniff');
    expect(DASHBOARD_SECURITY_HEADERS['X-Frame-Options']).toBe('DENY');
    expect(DASHBOARD_SECURITY_HEADERS['Referrer-Policy']).toBe('no-referrer');
  });
});
