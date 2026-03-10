/**
 * PicoClaw Whiteboard API
 * Provides programmatic control for AI agent interaction via PinchTab
 */

(function() {
  'use strict';

  window.picoclaw = window.picoclaw || {};
  
  // Store reference to the drawingboard instance
  let board = null;
  
  /**
   * Initialize the API with the drawingboard instance
   * Call this after the board is created
   */
  window.picoclaw.init = function(drawingBoard) {
    board = drawingBoard;
    console.log('[PicoClaw API] Initialized');
  };
  
  /**
   * Draw a highlight rectangle on the canvas
   * @param {number} x - X coordinate (0-1, relative to canvas width)
   * @param {number} y - Y coordinate (0-1, relative to canvas height)
   * @param {number} width - Width (0-1, relative to canvas width)
   * @param {number} height - Height (0-1, relative to canvas height)
   * @param {string} color - Color (hex or named color, default: red)
   * @param {number} lineWidth - Line width in pixels (default: 3)
   */
  window.picoclaw.highlightRect = function(x, y, width, height, color, lineWidth) {
    if (!board) {
      console.error('[PicoClaw API] Board not initialized');
      return { success: false, error: 'Board not initialized' };
    }
    
    color = color || '#ff0000';
    lineWidth = lineWidth || 3;
    
    const canvas = board.canvas;
    const ctx = board.ctx;
    const canvasWidth = canvas.width;
    const canvasHeight = canvas.height;
    
    // Convert relative coordinates to absolute
    const absX = x * canvasWidth;
    const absY = y * canvasHeight;
    const absWidth = width * canvasWidth;
    const absHeight = height * canvasHeight;
    
    // Save current state
    const prevColor = board.color;
    const prevSize = board.size;
    
    // Set highlight style
    ctx.strokeStyle = color;
    ctx.lineWidth = lineWidth;
    ctx.strokeRect(absX, absY, absWidth, absHeight);
    
    // Restore state
    board.color = prevColor;
    board.size = prevSize;
    
    // Save to history
    board.saveHistory();
    
    console.log(`[PicoClaw API] Drew rectangle at (${x}, ${y}) size (${width}, ${height})`);
    return { success: true, x, y, width, height, color };
  };
  
  /**
   * Draw a circle highlight
   * @param {number} x - Center X (0-1, relative)
   * @param {number} y - Center Y (0-1, relative)
   * @param {number} radius - Radius (0-1, relative to canvas width)
   * @param {string} color - Color
   * @param {number} lineWidth - Line width
   */
  window.picoclaw.highlightCircle = function(x, y, radius, color, lineWidth) {
    if (!board) {
      console.error('[PicoClaw API] Board not initialized');
      return { success: false, error: 'Board not initialized' };
    }
    
    color = color || '#ff0000';
    lineWidth = lineWidth || 3;
    
    const canvas = board.canvas;
    const ctx = board.ctx;
    const canvasWidth = canvas.width;
    const canvasHeight = canvas.height;
    
    const absX = x * canvasWidth;
    const absY = y * canvasHeight;
    const absRadius = radius * canvasWidth;
    
    ctx.strokeStyle = color;
    ctx.lineWidth = lineWidth;
    ctx.beginPath();
    ctx.arc(absX, absY, absRadius, 0, 2 * Math.PI);
    ctx.stroke();
    
    board.saveHistory();
    
    console.log(`[PicoClaw API] Drew circle at (${x}, ${y}) radius ${radius}`);
    return { success: true, x, y, radius, color };
  };
  
  /**
   * Add text annotation
   * @param {string} text - Text to display
   * @param {number} x - X position (0-1, relative)
   * @param {number} y - Y position (0-1, relative)
   * @param {string} color - Text color
   * @param {number} fontSize - Font size in pixels (default: 16)
   */
  window.picoclaw.addText = function(text, x, y, color, fontSize) {
    if (!board) {
      console.error('[PicoClaw API] Board not initialized');
      return { success: false, error: 'Board not initialized' };
    }
    
    color = color || '#ff0000';
    fontSize = fontSize || 16;
    
    const canvas = board.canvas;
    const ctx = board.ctx;
    const canvasWidth = canvas.width;
    const canvasHeight = canvas.height;
    
    const absX = x * canvasWidth;
    const absY = y * canvasHeight;
    
    ctx.fillStyle = color;
    ctx.font = `${fontSize}px Arial`;
    ctx.fillText(text, absX, absY);
    
    board.saveHistory();
    
    console.log(`[PicoClaw API] Added text "${text}" at (${x}, ${y})`);
    return { success: true, text, x, y, color, fontSize };
  };
  
  /**
   * Clear the entire canvas
   */
  window.picoclaw.clear = function() {
    if (!board) {
      console.error('[PicoClaw API] Board not initialized');
      return { success: false, error: 'Board not initialized' };
    }
    
    board.reset();
    console.log('[PicoClaw API] Canvas cleared');
    return { success: true };
  };
  
  /**
   * Get canvas data URL for screenshot/analysis
   * @param {string} format - Image format (default: 'image/png')
   */
  window.picoclaw.getDataURL = function(format) {
    if (!board) {
      console.error('[PicoClaw API] Board not initialized');
      return { success: false, error: 'Board not initialized' };
    }
    
    format = format || 'image/png';
    const dataURL = board.canvas.toDataURL(format);
    
    console.log('[PicoClaw API] Generated data URL');
    return { success: true, dataURL };
  };
  
  /**
   * Get current canvas state info
   */
  window.picoclaw.getState = function() {
    if (!board) {
      return { initialized: false };
    }
    
    return {
      initialized: true,
      width: board.canvas.width,
      height: board.canvas.height,
      color: board.color,
      size: board.size
    };
  };
  
  console.log('[PicoClaw API] Loaded and ready');
})();
