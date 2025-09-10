import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@/test/utils/test-utils';
import userEvent from '@testing-library/user-event';
import { AssignmentModal } from '../AssignmentModal';

const mockRulePacks = [
  {
    id: '550e8400-e29b-41d4-a716-446655440001',
    name: 'Security Policy',
    description: 'Basic security rules',
    currentVersionId: 'v1',
    isActive: true,
    createdAt: '2024-01-01T00:00:00Z',
    updatedAt: '2024-01-01T00:00:00Z',
  },
  {
    id: '550e8400-e29b-41d4-a716-446655440002',
    name: 'Content Filter',
    description: 'Content filtering rules',
    currentVersionId: 'v1',
    isActive: true,
    createdAt: '2024-01-01T00:00:00Z',
    updatedAt: '2024-01-01T00:00:00Z',
  },
];

describe('AssignmentModal', () => {
  const mockOnSubmit = vi.fn();
  const mockOnClose = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders modal when open', () => {
    render(
      <AssignmentModal
        isOpen={true}
        onClose={mockOnClose}
        onSubmit={mockOnSubmit}
        rulePacks={mockRulePacks}
        isLoading={false}
      />
    );

    expect(screen.getByText(/Assign RulePack/i)).toBeInTheDocument();
    expect(screen.getByTestId('select-rulepack')).toBeInTheDocument();
    expect(screen.getByTestId('input-endpoint-chip')).toBeInTheDocument();
    expect(screen.getByText('Priority')).toBeInTheDocument();
  });

  it('does not render when closed', () => {
    render(
      <AssignmentModal
        isOpen={false}
        onClose={mockOnClose}
        onSubmit={mockOnSubmit}
        rulePacks={mockRulePacks}
        isLoading={false}
      />
    );

    expect(screen.queryByText('Assign RulePack')).not.toBeInTheDocument();
  });

  it('populates rulepack select with provided rulepacks', async () => {
    const user = userEvent.setup({ pointerEventsCheck: 0 });
    render(
      <AssignmentModal
        isOpen={true}
        onClose={mockOnClose}
        onSubmit={mockOnSubmit}
        rulePacks={mockRulePacks}
        isLoading={false}
      />
    );

    const rulepackSelect = screen.getByTestId('select-rulepack');
    await user.click(rulepackSelect);

    expect(await screen.findByText('Security Policy')).toBeInTheDocument();
    expect(await screen.findByText('Content Filter')).toBeInTheDocument();
  });

  it('allows adding endpoints via chip input', async () => {
    const user = userEvent.setup({ pointerEventsCheck: 0 });
    render(
      <AssignmentModal
        isOpen={true}
        onClose={mockOnClose}
        onSubmit={mockOnSubmit}
        rulePacks={mockRulePacks}
        isLoading={false}
      />
    );

    const endpointsInput = screen.getByTestId('input-endpoint-chip');
    await user.type(endpointsInput, '/api/v1/users');
    await user.keyboard('{Enter}');

    expect(screen.getAllByText('/api/v1/users').length).toBeGreaterThan(0);
  });

  it('allows removing endpoints', async () => {
    const user = userEvent.setup({ pointerEventsCheck: 0 });
    render(
      <AssignmentModal
        isOpen={true}
        onClose={mockOnClose}
        onSubmit={mockOnSubmit}
        rulePacks={mockRulePacks}
        isLoading={false}
      />
    );

    const endpointsInput = screen.getByTestId('input-endpoint-chip');
    await user.type(endpointsInput, '/api/v1/users');
    await user.keyboard('{Enter}');

    const removeButtons = screen.getAllByRole('button');
    await user.click(removeButtons[removeButtons.length - 1]);

    expect(screen.queryByText('/api/v1/users')).not.toBeInTheDocument();
  });

  it('populates priority select with correct options', async () => {
    const user = userEvent.setup({ pointerEventsCheck: 0 });
    render(
      <AssignmentModal
        isOpen={true}
        onClose={mockOnClose}
        onSubmit={mockOnSubmit}
        rulePacks={mockRulePacks}
        isLoading={false}
      />
    );

    const prioritySelect = screen.getByTestId('select-priority');
    await user.click(prioritySelect);

    expect(await screen.findAllByText('High')).not.toHaveLength(0);
    expect(await screen.findAllByText('Medium')).not.toHaveLength(0);
    expect(await screen.findAllByText('Low')).not.toHaveLength(0);
  });

  it('submits form with correct data', async () => {
    const user = userEvent.setup({ pointerEventsCheck: 0 });
    render(
      <AssignmentModal
        isOpen={true}
        onClose={mockOnClose}
        onSubmit={mockOnSubmit}
        rulePacks={mockRulePacks}
        isLoading={false}
      />
    );

    const rulepackSelect = screen.getByTestId('select-rulepack');
    await user.click(rulepackSelect);
    // Use keyboard to select first item reliably
    await user.keyboard('{ArrowDown}{Enter}');

    const endpointsInput = screen.getByTestId('input-endpoint-chip');
    await user.type(endpointsInput, '/api/v1/users');
    await user.keyboard('{Enter}');
    await user.type(endpointsInput, '/api/v1/admin');
    await user.keyboard('{Enter}');

    const prioritySelect = screen.getByTestId('select-priority');
    await user.click(prioritySelect);
    // Default is Medium; move to High via keyboard (up once) and confirm
    await user.keyboard('{ArrowUp}{Enter}');

    const submitButton = screen.getByTestId('button-submit-assignment');
    await user.click(submitButton);

    await waitFor(() => expect(mockOnSubmit).toHaveBeenCalledTimes(2));
    const calls = (mockOnSubmit as any).mock.calls;
    const first = calls[0][0];
    const second = calls[1][0];
    expect([mockRulePacks[0].id, mockRulePacks[1].id]).toContain(first.rulepackId);
    expect(second.rulepackId).toBe(first.rulepackId);
    expect(first.targetScope).toBe('/api/v1/users');
    expect(second.targetScope).toBe('/api/v1/admin');
    expect(first.priority).toBe(750);
    expect(second.priority).toBe(750);
    expect(first.enabled).toBe(true);
    expect(second.enabled).toBe(true);
  });

  it('submits selected method in payload', async () => {
    const user = userEvent.setup({ pointerEventsCheck: 0 });
    render(
      <AssignmentModal
        isOpen={true}
        onClose={mockOnClose}
        onSubmit={mockOnSubmit}
        rulePacks={mockRulePacks}
        isLoading={false}
      />
    );

    // Select a rulepack
    await user.click(screen.getByTestId('select-rulepack'));
    await user.keyboard('{ArrowDown}{Enter}');

    // Select method POST via keyboard to avoid ambiguous matches in Radix portal
    await user.click(screen.getByTestId('select-method'));
    // Default is '*', so two ArrowDown presses -> GET then POST
    await user.keyboard('{ArrowDown}{ArrowDown}{Enter}');

    // Add a single endpoint
    const ep = screen.getByTestId('input-endpoint-chip');
    await user.type(ep, '/api/v1/items');
    await user.keyboard('{Enter}');

    // Submit
    await user.click(screen.getByTestId('button-submit-assignment'));

    await waitFor(() => expect(mockOnSubmit).toHaveBeenCalledTimes(1));
    const arg = (mockOnSubmit as any).mock.calls[0][0];
    expect(arg.method).toBe('POST');
    expect(arg.targetScope).toBe('/api/v1/items');
    expect([mockRulePacks[0].id, mockRulePacks[1].id]).toContain(arg.rulepackId);
  });

  it('validates required fields', async () => {
    const user = userEvent.setup({ pointerEventsCheck: 0 });
    render(
      <AssignmentModal
        isOpen={true}
        onClose={mockOnClose}
        onSubmit={mockOnSubmit}
        rulePacks={mockRulePacks}
        isLoading={false}
      />
    );

    const submitButton = screen.getByTestId('button-submit-assignment');
    await user.click(submitButton);
    // Should not submit and fields should be marked invalid
    await waitFor(() => {
      expect(screen.getByTestId('select-rulepack')).toHaveAttribute('aria-invalid', 'true');
      expect(screen.getByTestId('input-endpoint-chip')).toHaveAttribute('aria-invalid', 'true');
      expect(mockOnSubmit).not.toHaveBeenCalled();
    });
  });

  it('shows loading state', () => {
    render(
      <AssignmentModal
        isOpen={true}
        onClose={mockOnClose}
        onSubmit={mockOnSubmit}
        rulePacks={mockRulePacks}
        isLoading={true}
      />
    );

    const submitButton = screen.getByTestId('button-submit-assignment');
    expect(submitButton).toBeDisabled();
  });

  it('calls onClose when cancel button is clicked', async () => {
    const user = userEvent.setup({ pointerEventsCheck: 0 });
    render(
      <AssignmentModal
        isOpen={true}
        onClose={mockOnClose}
        onSubmit={mockOnSubmit}
        rulePacks={mockRulePacks}
        isLoading={false}
      />
    );

    const cancelButtons = screen.getAllByRole('button');
    await user.click(cancelButtons[0]);

    expect(mockOnClose).toHaveBeenCalled();
  });
});
