import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@/test/utils/test-utils';
import userEvent from '@testing-library/user-event';
import { setupAuth, clearAuth } from '@/test/utils/test-utils';
import PolicyAssignments from '../PolicyAssignments';

// Mock the API modules
vi.mock('@/lib/api', () => ({
  policyAssignmentApi: {
    getAll: vi.fn(),
    create: vi.fn(),
    batchCreate: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
  },
  rulePackApi: {
    getAll: vi.fn(),
  },
}));

vi.mock('@/hooks/use-toast', () => ({
  useToast: () => ({
    toast: vi.fn(),
  }),
}));

describe('PolicyAssignments', () => {
  const mockAssignments = [
    {
      id: '123e4567-e89b-12d3-a456-426614174000',
      rulepackId: '550e8400-e29b-41d4-a716-446655440001',
      targetScope: '/api/v1/users',
      endpoints: ['/api/v1/users', '/api/v1/admin'],
      priority: 100,
      enabled: true,
      createdAt: '2024-01-01T00:00:00Z',
      updatedAt: '2024-01-01T00:00:00Z',
    },
  ];

  const mockRulePacks = [
    {
      id: '550e8400-e29b-41d4-a716-446655440001',
      name: 'Security Policy',
      description: 'Basic security rules',
      currentVersionId: 'v1',
      createdAt: '2024-01-01T00:00:00Z',
      updatedAt: '2024-01-01T00:00:00Z',
    },
  ];

  beforeEach(() => {
    vi.clearAllMocks();
    clearAuth();
    setupAuth();
  });

  it('renders assignments table with data', async () => {
    const { policyAssignmentApi, rulePackApi } = await import('@/lib/api');
    (policyAssignmentApi.getAll as any).mockResolvedValue({
      data: mockAssignments,
      total: 1,
    });
    (rulePackApi.getAll as any).mockResolvedValue({
      data: mockRulePacks,
      total: 1,
    });

    render(<PolicyAssignments />);

    await waitFor(() => {
      expect(screen.getByText('Security Policy')).toBeInTheDocument();
      expect(screen.getByText('/api/v1/users')).toBeInTheDocument();
      expect(screen.getByText('100')).toBeInTheDocument();
    });
  });

  it('shows method badge and defaults to Any (*) when method is missing', async () => {
    const { policyAssignmentApi, rulePackApi } = await import('@/lib/api');
    (policyAssignmentApi.getAll as any).mockResolvedValue({ data: mockAssignments, total: 1 });
    (rulePackApi.getAll as any).mockResolvedValue({ data: mockRulePacks, total: 1 });

    render(<PolicyAssignments />);

    const methodBadge = await screen.findByTestId('assignment-method-123e4567-e89b-12d3-a456-426614174000');
    expect(methodBadge).toHaveTextContent('Any (*)');
  });

  it('shows empty state when no assignments', async () => {
    const { policyAssignmentApi, rulePackApi } = await import('@/lib/api');
    (policyAssignmentApi.getAll as any).mockResolvedValue({
      data: [],
      total: 0,
    });
    (rulePackApi.getAll as any).mockResolvedValue({
      data: mockRulePacks,
      total: 1,
    });

    render(<PolicyAssignments />);

    await waitFor(() => {
      expect(screen.getByText('No RulePack assignments found')).toBeInTheDocument();
    });
  });

  it('opens create assignment modal when button is clicked', async () => {
    const user = userEvent.setup({ pointerEventsCheck: 0 });
    const { policyAssignmentApi, rulePackApi } = await import('@/lib/api');
    (policyAssignmentApi.getAll as any).mockResolvedValue({
      data: [],
      total: 0,
    });
    (rulePackApi.getAll as any).mockResolvedValue({
      data: mockRulePacks,
      total: 1,
    });

    render(<PolicyAssignments />);

    await user.click(await screen.findByText('Assign RulePack'));

    expect(await screen.findByTestId('select-rulepack')).toBeInTheDocument();
    expect(await screen.findByTestId('input-endpoint-chip')).toBeInTheDocument();
  });

  it('creates new assignment successfully', async () => {
    const user = userEvent.setup({ pointerEventsCheck: 0 });
    const { policyAssignmentApi, rulePackApi } = await import('@/lib/api');
    (policyAssignmentApi.getAll as any).mockResolvedValue({
      data: [],
      total: 0,
    });
    (rulePackApi.getAll as any).mockResolvedValue({
      data: mockRulePacks,
      total: 1,
    });

    (policyAssignmentApi.batchCreate as any).mockResolvedValue({ created: [] });

    render(<PolicyAssignments />);

    await user.click(await screen.findByText('Assign RulePack'));

    const rulepackSelect = await screen.findByTestId('select-rulepack');
    await user.click(rulepackSelect);
    // Radix renders options as spans; use findAllByText and click the first match
    const option = await screen.findAllByText('Security Policy');
    await user.click(option[0]);

    const endpointsInput = screen.getByTestId('input-endpoint-chip');
    await user.type(endpointsInput, '/api/v1/test');
    await user.keyboard('{Enter}');

    const submitButton = screen.getByTestId('button-submit-assignment');
    await user.click(submitButton);

    await waitFor(() => {
      expect(policyAssignmentApi.batchCreate).toHaveBeenCalled();
    });
  });

  it('propagates selected method from modal to policyAssignmentApi.create', async () => {
    const user = userEvent.setup({ pointerEventsCheck: 0 });
    const { policyAssignmentApi, rulePackApi } = await import('@/lib/api');
    (policyAssignmentApi.getAll as any).mockResolvedValue({ data: [], total: 0 });
    (rulePackApi.getAll as any).mockResolvedValue({ data: mockRulePacks, total: 1 });
    (policyAssignmentApi.batchCreate as any).mockResolvedValue({ created: [] });

    render(<PolicyAssignments />);

    // Open modal
    await user.click(await screen.findByText('Assign RulePack'));

    // Pick rulepack
    await user.click(await screen.findByTestId('select-rulepack'));
    const rulepackOption = await screen.findAllByText('Security Policy');
    await user.click(rulepackOption[0]);

    // Pick method = POST via keyboard to avoid ambiguous matches
    await user.click(screen.getByTestId('select-method'));
    await user.keyboard('{ArrowDown}{ArrowDown}{Enter}');

    // Add endpoint
    const ep = screen.getByTestId('input-endpoint-chip');
    await user.type(ep, '/api/v1/create');
    await user.keyboard('{Enter}');

    // Submit
    await user.click(screen.getByTestId('button-submit-assignment'));

    await waitFor(() => {
      expect(policyAssignmentApi.batchCreate).toHaveBeenCalledTimes(1);
      const arg = (policyAssignmentApi.batchCreate as any).mock.calls[0][0];
      expect(Array.isArray(arg)).toBe(true);
      expect(arg[0].method).toBe('POST');
      expect(arg[0].targetScope).toBe('/api/v1/create');
    });
  });

  it('filters assignments by method and uses correct badge styling', async () => {
    const user = userEvent.setup({ pointerEventsCheck: 0 });
    const { policyAssignmentApi, rulePackApi } = await import('@/lib/api');
    const assignments = [
      {
        id: 'a1', rulepackId: 'rp1', targetScope: '/get-only', method: 'GET', priority: 100, enabled: true,
        createdAt: '2024-01-01T00:00:00Z', updatedAt: '2024-01-01T00:00:00Z'
      },
      {
        id: 'a2', rulepackId: 'rp2', targetScope: '/post-only', method: 'POST', priority: 100, enabled: true,
        createdAt: '2024-01-01T00:00:00Z', updatedAt: '2024-01-01T00:00:00Z'
      },
      {
        id: 'a3', rulepackId: 'rp3', targetScope: '/any-match', method: '*', priority: 100, enabled: true,
        createdAt: '2024-01-01T00:00:00Z', updatedAt: '2024-01-01T00:00:00Z'
      },
    ];
    const rps = [
      { id: 'rp1', name: 'RP1' },
      { id: 'rp2', name: 'RP2' },
      { id: 'rp3', name: 'RP3' },
    ];
    (policyAssignmentApi.getAll as any).mockResolvedValue({ data: assignments, total: 3 });
    (rulePackApi.getAll as any).mockResolvedValue({ data: rps, total: 3 });

    render(<PolicyAssignments />);

    // Select GET filter using keyboard (order: All -> Any (*) -> GET)
    await user.click(await screen.findByTestId('select-method-filter'));
    await user.keyboard('{ArrowDown}{ArrowDown}{Enter}');

    // Expect only GET row visible
    await waitFor(() => {
      expect(screen.getByText('/get-only')).toBeInTheDocument();
      expect(screen.queryByText('/post-only')).not.toBeInTheDocument();
      expect(screen.queryByText('/any-match')).not.toBeInTheDocument();
    });

    // Select Any (*) filter via keyboard (open and ArrowDown once from GET to Any (*))
    await user.click(screen.getByTestId('select-method-filter'));
    await user.keyboard('{ArrowUp}{Enter}');

    await waitFor(() => {
      expect(screen.getByText('/any-match')).toBeInTheDocument();
      expect(screen.queryByText('/get-only')).not.toBeInTheDocument();
      expect(screen.queryByText('/post-only')).not.toBeInTheDocument();
    });

    // Back to All Methods (open and ArrowUp from Any (*) -> All Methods)
    await user.click(screen.getByTestId('select-method-filter'));
    await user.keyboard('{ArrowUp}{Enter}');

    await waitFor(() => {
      expect(screen.getByText('/get-only')).toBeInTheDocument();
      expect(screen.getByText('/post-only')).toBeInTheDocument();
      expect(screen.getByText('/any-match')).toBeInTheDocument();
    });

    // Badge styling assertions
    const getBadge = await screen.findByTestId('assignment-method-a1');
    const anyBadge = await screen.findByTestId('assignment-method-a3');
    expect(getBadge).toHaveTextContent('GET');
    expect(anyBadge).toHaveTextContent('Any (*)');
  });

  it('secures arbitrary endpoint input by creating assignment for custom path', async () => {
    const user = userEvent.setup({ pointerEventsCheck: 0 });
    const { policyAssignmentApi, rulePackApi } = await import('@/lib/api');
    (policyAssignmentApi.getAll as any).mockResolvedValue({ data: [], total: 0 });
    (rulePackApi.getAll as any).mockResolvedValue({ data: mockRulePacks, total: 1 });
    (policyAssignmentApi.create as any).mockResolvedValue({ id: 'new' });

    render(<PolicyAssignments />);

    await user.click(await screen.findByText('Assign RulePack'));
    await user.click(await screen.findByTestId('select-rulepack'));
    const rpOpt = await screen.findAllByText('Security Policy');
    await user.click(rpOpt[0]);

    const ep = screen.getByTestId('input-endpoint-chip');
    await user.type(ep, '/custom/endpoint');
    await user.keyboard('{Enter}');

    await user.click(screen.getByTestId('button-submit-assignment'));

    await waitFor(() => {
      expect(policyAssignmentApi.batchCreate).toHaveBeenCalled();
      const arg = (policyAssignmentApi.batchCreate as any).mock.calls[0][0];
      expect(Array.isArray(arg)).toBe(true);
      expect(arg[0].targetScope).toBe('/custom/endpoint');
    });
  });
});
